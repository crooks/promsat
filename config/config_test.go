package config

import (
	"io/ioutil"
	"os"
	"testing"
)

func TestFlags(t *testing.T) {
	expectedConfig := "/etc/promsat/fake.yml"
	// This needs to be set prior to doing ParseFlags()
	os.Setenv("PROMSATCFG", expectedConfig)
	f := ParseFlags()
	if f.Config != expectedConfig {
		t.Fatalf("Expected --config to contain \"%v\" but got \"%v\".", expectedConfig, f.Config)
	}
}

func TestConfig(t *testing.T) {
	testFile, err := ioutil.TempFile("/tmp", "yamn")
	if err != nil {
		t.Fatalf("Unable to create TempFile: %v", err)
	}
	//defer os.Remove(testFile.Name())
	fakeCfg := new(Config)
	fakeCfg.ExporterJob = "fake_exporter"
	fakeCfg.Labels = map[string]string{
		"env": "fake",
	}
	fakeCfg.AutoLabel = "env"
	fakeCfg.WriteConfig(testFile.Name())

	cfg, err := ParseConfig(testFile.Name())
	if err != nil {
		t.Fatalf("ParseConfig returned: %v", err)
	}

	if cfg.ExporterJob != fakeCfg.ExporterJob {
		t.Fatalf("Expected cfg.ExporterJob to contain \"%v\" but got \"%v\".", fakeCfg.ExporterJob, cfg.ExporterJob)
	}
}
