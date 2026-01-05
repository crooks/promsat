package main

import (
	"testing"
)

func TestHostName(t *testing.T) {
	var tests = []struct {
		input         string
		defaultDomain string
		shortName     string
		fqdnName      string
	}{
		{"foo.testdomain.com", "fakedomain.com", "foo", "foo.testdomain.com"},
		{"foo.testdomain.com", "testdomain.com", "foo", "foo.testdomain.com"},
		{"foo", "fakedomain.com", "foo", "foo.fakedomain.com"},
		{"foo..testdomain.com", "testdomain.com", "foo", "foo.testdomain.com"},
		{"foo..testdomain.com", "fakedomain.com", "foo", "foo.testdomain.com"},
		{"foo", "bar", "foo", "foo.bar"},
	}

	for n, tt := range tests {
		shortName, fqdnName := fullyQualify(tt.input, tt.defaultDomain)
		if shortName != tt.shortName {
			t.Errorf("Test %d failed: Unexpected shortName: Wanted=%s, Got=%s", n, tt.shortName, shortName)
		}
		if fqdnName != tt.fqdnName {
			t.Errorf("Test %d failed: Unexpected fqdnName: Wanted=%s, Got=%s", n, tt.fqdnName, fqdnName)
		}
	}
}
