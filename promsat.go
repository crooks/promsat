package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"strings"

	"github.com/crooks/promsat/config"
	"github.com/tidwall/gjson"
)

// existingTargets is a slice of targets already known to Prometheus (exclulding auto-added).
type existingTargets struct {
	hosts []string
}

// autoTargets is only used to make JSON Marshal wrap autoTarget in an array.
type autoTargets []*autoTarget

// Each section of a prometheus Service Description file is make up of Labels and Targets.  These (all one of them)
// will be appended to autoTargets.
type autoTarget struct {
	Labels  map[string]string `json:"labels"`
	Targets []string          `json:"targets"`
}

// contains returns true if slice contains str.
func contains(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}

// newAutoTarget returns an instance of autoTarget
func newAutoTarget() *autoTarget {
	return &autoTarget{
		Labels:  make(map[string]string),
		Targets: make([]string, 0),
	}
}

// jsonFromFile takes the filename for a file containing json formatted content
// and returns a gjson Result of the file content.
func jsonFromFile(filename string) (gjson.Result, error) {
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return gjson.Result{}, err
	}
	return gjson.ParseBytes(b), nil
}

// writeTargets outputs a Prometheus config to file
func (at *autoTarget) writeTargets(filename string) (err error) {
	// Although we are only writing a single entry to the targets file, Prometheus expects it to be in a list.
	targets := make(autoTargets, 1)
	for k, v := range cfg.Labels {
		at.Labels[k] = v
	}
	targets[0] = at
	b, _ := json.MarshalIndent(targets, "", "  ")
	err = ioutil.WriteFile(filename, b, 0644)
	if err != nil {
		return
	}
	return
}

// shortName take a hostname string and returns the shortname for it.
func shortName(host string) string {
	return strings.Split(host, ".")[0]
}

// compareToSat takes a slice of known target hosts and compares it to defined Satellite hosts.  It returns a slice of
// hosts that are in Satellite but are not Prometheus targets.
func (t *existingTargets) compareToSat() *autoTarget {
	j, err := jsonFromFile("/home/crooks/sample_json/rhsat.json")
	if err != nil {
		log.Fatalf("Unable to parse json file: %v", err)
	}
	at := newAutoTarget()
	for _, v := range j.Get("results").Array() {
		host := v.Get("name")
		ip := v.Get("ip")
		subscription := v.Get("subscription_status")
		if !host.Exists() || !subscription.Exists() || !ip.Exists() {
			// To be considered valid, a Satellite host must have a name, a subscription and an IP address.
			continue
		}
		short := shortName(host.String())
		if subscription.Int() != 0 {
			log.Printf("Invalid subscription for %s", short)
			continue
		}
		if ip.String() == "" {
			log.Printf("No IPv4 address for %s", short)
			continue
		}
		// Don't add hosts that are explicitly excluded
		if contains(cfg.ExcludeHosts, short) {
			log.Printf("Host %s is excluded", short)
			continue
		}

		// Is this Satellite host already known to Prometheus?
		if contains(t.hosts, short) {
			continue
		}
		shortPort := fmt.Sprintf("%s:%d", short, cfg.AutoPort)
		at.Targets = append(at.Targets, shortPort)
	}
	return at
}

func (t *existingTargets) getPrometheusTargets() (err error) {
	j, err := jsonFromFile("/home/crooks/sample_json/prom.json")
	if err != nil {
		log.Fatalf("Unable to parse json file: %v", err)
	}
	for _, target := range j.Get("data.activeTargets").Array() {
		labels := target.Get("labels")
		job := labels.Get("job")
		instance := labels.Get("instance")
		// This is a bit kludgy!  Targets written to the auto file (in a previous run) will now be defined targets.
		// As a consequence they will be treated as defined targets (in the current run) and will not be included in the
		// new auto_targets file.  The solution is to exclude them from the slice of known targets.
		autoLabel := labels.Get(cfg.AutoLabel)
		if autoLabel.Exists() && autoLabel.String() == cfg.Labels[cfg.AutoLabel] {
			continue
		}
		if instance.Exists() && job.Exists() && job.String() == cfg.NodeExporter {
			t.hosts = append(t.hosts, instance.String())
			log.Printf("Prometheus knows about: %s", instance)
		}
	}
	if len(t.hosts) == 0 {
		err = errors.New("zero targets returned")
		return
	}
	log.Printf("Prometheus targets found: %d", len(t.hosts))
	return
}

var (
	cfg *config.Config
)

func main() {
	var err error
	flags := config.ParseFlags()
	if !flags.Debug {
		log.SetOutput(ioutil.Discard)
	}
	cfg, err = config.ParseConfig(flags.Config)
	if err != nil {
		log.Fatalf("Cannot parse config: %v", err)
	}
	// Create a slice for targets discovered in Prometheus
	t := new(existingTargets)
	t.hosts = make([]string, 0)
	err = t.getPrometheusTargets()
	if err != nil {
		log.Fatalf("Unable to parse Prometheus API: %v", err)
	}
	at := t.compareToSat()
	at.writeTargets(cfg.OutJSON)
}
