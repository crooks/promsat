## Introduction
Promsat is intended to maintain a list of Prometheus targets based on server information pulled from Red Hat Satellite. A hosts list is pulled from the Satellite API and compared with a similar list pulled from the Prometheus API.  If a host is known to Satellite but is not a defined target in Prometheus, it is added to a dynamically maintained targets file.

## Installation
### Install promsat
* Make sure you have [Go](https://go.dev) installed and working
* Grab the Go code from the [Github Repository](https://github.com/crooks/promsat)
* Compile the code with something like `go build` and copy the resulting binary to somewhere sane. E.g. /usr/local/bin/promsat
### Configure promsat
Promsat requires a YAML formatted configuration file.  The location of the file is defined when the binary is executed:-

`promsat --config /etc/prometheus/promsat.yml`

The config file should look something like this:-

```yaml
api_user: satellite_username
apt_password: satellite_password
baseurl_satellite: https://sat.domain.com
baseurl_prometheus: https://prometheus.domain.com
exporter_job: node_exporter
target_filename: /var/local/cache/prometheus/auto_targets.json
labels:
  env: auto
  notify: unixteam
autohosts_label: env
autohosts_port: 9100
exclude_hosts:
  - dontmonitorme
exclude_host_prefix:
  - excludeme
```

Where:
* api_user = Username for accessing the Satellite API
* api_password = Password for Satellite API user (A low privilege, read-only account)
* baseurl_satellite = The URL of the Satellite server hosting the API
* baseurl_prometheus = The URL of the Prometheus server
* exporter_job = Name of the reporter job associated with this auto-discovery process
* target_filename = Filename where auto-discovered targets should be written (This should be added to the Node Exporter `file_sd_config` targets list)
* labels = Labels to append to each auto-discovered host
* autohosts_label = A label that uniquely identifies auto-discovered hosts
* autohosts_port = Port number of the exporter associated with the auto-discovery process
* exclude_hosts = List of hostnames to exclude from the targets file
* exclude_host_prefix = As `exclude_hosts` but acts like a wildcard