# Oxcross

Oxcross is a simple HTTP latency monitoring system, allowing multiple edge servers to be monitored from multiple locations at the same time, with a centralised configuration mechanism. The system consists of:

* **`oxcross-origin`**, which runs as a systemd daemon on edge servers to be monitored. 
  * These daemons are optional. Any existing health check endpoint returning 200s can be monitored in the `simple` mode, at the loss of some synchronization information.
* **`oxcross-leaf`**, which runs as a systemd daemon on monitoring servers, and are responsible for probing server for HTTP response timing and perceived clock drifts (`advanced` mode only), and record these as metrics.
  * It is light-weight and can run on virtual server environments with minimal specs.
* **`configserver`**, which is responsible for distributing information of _origin_ servers and global configurations to _leaf_ clients. 
  * It is Docker-packed and ready for running in a Kubernetes cluster.

## Background

Through long-lasting bargain hunting, I have a small fleet of virtual private servers from budget hosting providers costing under $1/month distributed around the world. These servers are low-powered, and their management interfaces are very different to each other. Therefore, I could not [join them into a Kubernetes cluster](https://blog.scy.email/running-a-low-cost-distributed-kubernetes-cluster-on-bare-metal-with-wireguard.html) efficiently, and it has been difficult for me to make good uses of them.

Because the large variety of network environments these servers sit in, they can be very useful for testing HTTP round trip latency and other connectivity information from around the world. Under this model, there will be one `oxcross-origin` server for each location my Kubernetes cluster operates; and one `oxcross-leaf` monitor server for each location I have a low-powered server needing to put into use. Through a config distributed by `configserver` periodically reloaded by each `oxcross-leaf`, all leaves can monitor all origins, and export their results as [Prometheus](https://prometheus.io/docs/prometheus/latest/configuration/configuration/) metrics. All I then need to do is to scrape these metrics from my cluster and visualise them. 

However, this is only a good use case if I won't need to manually reconfigure each existing monitor server every time I add an edge server to be monitored. Furthermore, having many monitor servers won't be useful unless their results can be gathered at a centralised location easily. Struggling to find an existing solution fitting these requirements, I decided to write one.

![Oxcross system layout](https://images.ebornet.com/uploads/big/a47c229f94ad46e80ed627b1a5079f74.png)

## Configuration

Follow the order of `oxcross-origin` on servers to be monitored, then `configserver` for distributing configuration pointing to servers to be monitored, and finally `oxcross-leaf` on servers used for monitoring.

### `oxcross-origin`

This component is optional if you already have a health check endpoint which returns 200 responses. `oxcross-origin` will also do this and also exports some optional timing information for leaves to use.

To set it up on a node to be monitored:
```
apt install sudo git
git clone https://github.com/icydoge/Oxcross.git
cd Oxcross
sh setup_origin.sh
```

### `configserver`

Follow the example of [`config.yaml.example`](https://github.com/icydoge/Oxcross/blob/master/config.yaml.example), add all origin server locations into a `JSON` config file.
* In `simple` mode, Oxcross will send a GET request to `scheme://host:port/`, and monitor a 200 response.
* In `advanced` mode (`oxcross-origin` required), Oxcross will send a GET request to `scheme://host:port/oxcross` which exports timing informatin in a 200 response.

`configserver` is optimized for running in a Kubernetes cluster. If using Kubernetes:
* Wrap the JSON in a `ConfigMap` manifest as shown in [`config.yaml.example`](https://github.com/icydoge/Oxcross/blob/master/config.yaml.example)
* Apply the `ConfigMap` manifest to create in-cluster configuration
* Apply [`configserver.yaml`](https://github.com/icydoge/Oxcross/blob/master/configserver.yaml) to set up the `configserver`.
* The `Service` created (`go-oxcross-configserver.monitoring.svc.cluster.local`) will need to be fronted by some kind of load balancer or reverse proxy to be exposed to the internet.

If not running in a cluster:
* At project's root directory, run `GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-w -s" -o ./oxcross-configserver`
* Wrap the resulting binary `oxcross-configserver` in a daemon wrapper of your choice.
* Start the daemon

The binary will listen on `:9300` in either case.

### `oxcross-leaf`

This component does the actual monitoring. To set it up on a node and monitor origin nodes:
```
apt install sudo git
git clone https://github.com/icydoge/Oxcross.git
cd Oxcross
sh setup_leaf.sh <leaf-id> https://your-oxcross-configserver.example.com
```

You will need to give each leaf a unique `<leaf-id>` to identify it in metrics, and also supply the endpoint of your `configserver` available over the internet or some kind of transit link. The leaf will automatically retrieve config from `https://your-oxcross-configserver.example.com/config` and keep it up to date as you change the config from `configserver`'s end.

## Metrics

`oxcross-leaf` instances export Prometheus metrics on `:9299`, which can be scraped through the internet or internal network by your Prometheus instance. An example Prometheus job can be found [here](https://github.com/icydoge/Oxcross/blob/master/prometheus.yaml.example).

The following metrics are available:
* `oxcross_leaf_probe_timings_{count|sum|bucket}`: a histogram counter providing HTTP round trip latency information from each leaf to each origin
* `oxcross_leaf_probe_results`: a success/fail counter allowing monitoring of reachability from each leaf to each origin
* `oxcross_leaf_origin_time_drift`: a timing gauge estimating the relative system time difference between each origin and each leaf which observed it. 

Once metrics are scraped, you can find an example Grafana dashboard JSON [here](https://github.com/icydoge/Oxcross/blob/master/grafana.json.example).

![Oxcross dashboard](https://images.ebornet.com/uploads/big/d40c193bd3d7c34b78b78ba0a747d5c9.png)

## TODOs

* `Prometheus` metrics are [low security-level](https://prometheus.io/docs/operating/security/) information. Therefore I haven't implemented TLS for metrics scraping. Due to the complexity of PKI management, this will have to be done later.
* Observe and export other types of useful connectivity information from `oxcross-leaf`, such as traceroute data.


