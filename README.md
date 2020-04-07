# Oxcross

Oxcross is a simple HTTP latency monitoring system for distributed deployment. It consists of:

* `origin` running as systemd daemons on servers to be monitored, listening on `:9301` facing the internet
* `leaf` running as systemd daemons on servers responsible for probing server for HTTP response timing and perceived clock drifts, and record these as metrics

**WIP: write systemd and metrics reporting**

