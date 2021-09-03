package main

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"strings"

	"github.com/crooks/promsat/config"
	"github.com/tidwall/gjson"
)

type prometheusTargets []*promTarget

type prometheusTarget struct {
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

func newPrometheusTarget() *prometheusTarget {
	return &prometheusTarget{
		Labels: make(map[string]string),
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

// writeSDConfig outputs a Prometheus config to file
func writeSDConfig(filename string, sd *promSDConfig) (err error) {
	// Although we are only writing a single entry to the targets file, Prometheus expects it to be in a list.
	sds := make([]promSDConfig, 1)
	for k, v := range cfg.Labels {
		sd.Labels[k] = v
	}
	sds[0] = *sd
	b, _ := json.MarshalIndent(sds, "", " ")
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
func (targets prometheusTargets) compareToSat() {
	j, err := jsonFromFile("/home/crooks/sample_json/rhsat.json")
	if err != nil {
		log.Fatalf("Unable to parse json file: %v", err)
	}
	sj := "{}"
	sj, err = sjson.Set(sj, "-1.labels", cfg.Labels)
	if err != nil {
		log.Fatalf("Unable to create array: %v", err)
	}
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
			log.Printf("Invalid subscription for %s", shortName)
			continue
		}
		if ip.String() == "" {
			log.Printf("No IPv4 address for %s", shortName)
			continue
		}
		if !contains(targets.Targets, short) {
			target.Targets = append(target.Targets, short)
		}
	}
}

func (targets prometheusTargets) getPrometheusTargets() (err error) {
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
		autoLabel := labels.Get(cfg.LabelAuto)
		if autoLabel.Exists() && autoLabel.String() == cfg.Labels[cfg.LabelAuto] {
			continue
		}
		if instance.Exists() && job.Exists() && job.String() == cfg.NodeExporter {
			targets = append(targets, instance.String())
		}
	}
	if len(targets) == 0 {
		err = errors.New("zero targets returned")
		return
	}
	return
}

var (
	cfg *config.Config
)

func main() {
	var err error
	flags := config.ParseFlags()
	cfg, err = config.ParseConfig(flags.Config)
	if err != nil {
		log.Fatalf("Cannot parse config: %v", err)
	}
	// Create a slice for targets discovered in Prometheus
	targets := newPrometheusTarget()
	err = t.getPrometheusTargets()
	if err != nil {
		log.Fatalf("Unable to parse Prometheus API: %v", err)
	}
	t.compareToSat()
}
