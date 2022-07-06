package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

	"github.com/crooks/promsat/api"
	"github.com/crooks/promsat/config"
	"github.com/tidwall/gjson"
)

var (
	cfg       *config.Config
	gitCommit string
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

// hasPrefix returns true if str has a prefix of any slice entry.
func hasPrefix(slice []string, str string) bool {
	for _, s := range slice {
		if strings.HasPrefix(str, s) {
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
func (at *autoTarget) writeTargets() (err error) {
	// Although we are only writing a single entry to the targets file, Prometheus expects it to be in a list.
	targets := make(autoTargets, 1)
	for k, v := range cfg.Labels {
		at.Labels[k] = v
	}
	targets[0] = at
	// Write targets to a temporary JSON file
	b, _ := json.MarshalIndent(targets, "", "  ")
	err = ioutil.WriteFile(cfg.OutJSONTmp, b, 0644)
	if err != nil {
		return
	}
	// Rename (overwrite) the temporary filename to the actual filename
	err = os.Rename(cfg.OutJSONTmp, cfg.OutJSON)
	if err != nil {
		err = fmt.Errorf("rename temp JSON failed: %v", err)
		return
	}
	return
}

// shortName take a hostname string and returns the shortname for it.
func shortName(host string) string {
	return strings.Split(host, ".")[0]
}

func getSatelliteHosts() gjson.Result {
	defer timeTrack(time.Now(), "getSatelliteHosts")
	authAPI := api.NewBasicAuthClient(cfg.APIUser, cfg.APIPassword, cfg.APICertFile)
	hostsURL := fmt.Sprintf("%s/api/v2/hosts?per_page=1000", cfg.BaseURLSatAPI)
	bytes, err := authAPI.GetJSON(hostsURL)
	if err != nil {
		// Little point in returning an error here.  No API is fatal.
		log.Fatalf("Unable to parse Satellite API: %v", err)
	}
	return gjson.ParseBytes(bytes)
}

func getPrometheusHosts() gjson.Result {
	defer timeTrack(time.Now(), "getPrometheusHosts")
	hostsURL := fmt.Sprintf("%s/api/v1/targets", cfg.BaseURLPromAPI)
	bytes, err := api.GetNoAuth(hostsURL)
	if err != nil {
		log.Fatalf("Unable to parse Prometheus API: %v", err)
	}
	return gjson.ParseBytes(bytes)
}

// compareToSat takes a slice of known target hosts and compares it to defined Satellite hosts.  It returns a slice of
// hosts that are in Satellite but are not Prometheus targets.
func (t *existingTargets) compareToSat() *autoTarget {
	/*
		// Useful for testing on hosts without API access
		j, err := jsonFromFile("/home/crooks/sample_json/rhsat.json")
		if err != nil {
			log.Fatalf("Unable to parse json file: %v", err)
		}
	*/
	j := getSatelliteHosts()
	at := newAutoTarget()
	for _, v := range j.Get("results").Array() {
		host := v.Get("name")
		// All hosts need a hostame.  This check should never match but, never say never!
		if !host.Exists() || len(host.String()) == 0 {
			log.Println("Invalid hostname for Satellite host")
			continue
		}
		short := shortName(host.String())
		subscription := v.Get("subscription_status")
		if !subscription.Exists() || subscription.Int() != 0 {
			log.Printf("Invalid subscription for %s", short)
			continue
		}
		// This check is intended to exclude Red Hat Satellite virthosts
		osid := v.Get("operatingsystem_id")
		if !osid.Exists() || osid.Int() == 0 {
			log.Printf("No valid OS ID found for %s", short)
		}
		// Don't add hosts that are explicitly excluded
		if contains(cfg.ExcludeHosts, short) {
			log.Printf("Host %s is excluded", short)
			continue
		}
		// As with the previous exclude but this time, any hosts that have a prefix of an entry in cfg.ExcludePrefix.
		if hasPrefix(cfg.ExcludePrefix, short) {
			log.Printf("Host %s is excluded (by prefix)", short)
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

// getPrometheusTargets queries the Prometheus API and constructs a list of existing targets.
func (t *existingTargets) getPrometheusTargets() {
	/*
		j, err := jsonFromFile("/home/crooks/sample_json/prom.json")
		if err != nil {
			log.Fatalf("Unable to parse json file: %v", err)
		}
	*/
	j := getPrometheusHosts()
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
		if instance.Exists() && job.Exists() && job.String() == cfg.ExporterJob {
			t.hosts = append(t.hosts, instance.String())
		}
	}
	if len(t.hosts) == 0 {
		log.Fatal("No prometheus targets found")
	}
	log.Printf("Prometheus targets found: %d", len(t.hosts))
}

// timeTrack can be used to time the processing duration of a function.
func timeTrack(start time.Time, name string) {
	elapsed := time.Since(start)
	log.Printf("%s took %s", name, elapsed)
}

func version() {
	fmt.Printf("Git Commit: %s\n", gitCommit)
	os.Exit(0)
}

func main() {
	var err error
	flags := config.ParseFlags()
	if flags.Version {
		version()
	}
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
	t.getPrometheusTargets()
	at := t.compareToSat()
	at.writeTargets()
}
