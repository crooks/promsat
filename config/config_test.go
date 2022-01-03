package config

import (
	"fmt"
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
	defer os.Remove(testFile.Name())
	fakeCfg := new(Config)
	fakeCfg.ExporterJob = "fake_exporter"
	fakeCfg.OutJSON = "/fake/file.json"
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

// tmpWriteRead is a small testing helper to write and then immediately read a config file
func (c *Config) tmpWriteRead(filename string) (cfg *Config, err error) {
	c.WriteConfig(filename)
	cfg, err = ParseConfig(filename)
	return
}

func TestBadConfig(t *testing.T) {
	testFile, err := ioutil.TempFile("/tmp", "yamn")
	if err != nil {
		t.Fatalf("Unable to create TempFile: %v", err)
	}
	defer os.Remove(testFile.Name())
	fakeCfg := new(Config)

	_, err = fakeCfg.tmpWriteRead(testFile.Name())
	if err != errUndefinedTargetFilename {
		t.Error("Undefined target_filename failed to return an error")
	}
	fakeCfg.OutJSON = "/fake/file.json"
	_, err = fakeCfg.tmpWriteRead(testFile.Name())
	if err != errUndefinedAutoHosts {
		t.Error("Undefined autohost_label failed to return an error")
	}
	fakeCfg.AutoLabel = "env"
	_, err = fakeCfg.tmpWriteRead(testFile.Name())
	if err != errUndefinedAutoLabel {
		t.Error("autohost_label is not specified in target_labels but expected error was not returned")
	}
	fakeCfg.Labels = map[string]string{
		"env": "fake",
	}
	_, err = fakeCfg.tmpWriteRead(testFile.Name())
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(fakeCfg)
}

func TestTmpFile(t *testing.T) {
	tmpFile := tmpFile("/a/test/dir/filename.foo")
	expectedTmpFile := "/a/test/dir/_filename.foo"
	if tmpFile != expectedTmpFile {
		t.Errorf("Unexpected temp filename. Expected=%s, Got=%s", expectedTmpFile, tmpFile)
	}

}
