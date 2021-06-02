# docker-compose

The docker-compose project runs a local optimism stack.

## prerequisites

- docker
- docker-compose
- make

## Starting and stopping the project

The base `docker-compose.yml` file will start the required components for a full stack.

Supplementing the base configuration is an additional metric enabling file, `docker-compose-metrics.yml`. Adding this configuration to the stack will enable metric emission for l2geth and start grafana (for metrics visualisation) and influxdb (for metric collection) instances.

Also available for testing is the `rpc-proxy` service in the `docker-compose-rpc-proxy.yml` file. It can be used to restrict what RPC methods are allowed to the Sequencer.

The base stack can be started and stopped with a command like this (there is no need to specify the default docker-compose.yml)
```
docker-compose \
    up --build --detach
```

To start the stack with monitoring enabled, just add the metric composition file.
```
docker-compose \
    -f docker-compose.yml \
    -f docker-compose-metrics.yml \
    up --build --detach
```

Optionally, run a verifier along the rest of the stack. Run a replica with the same command by switching the service name!
```
docker-compose up --scale \
    verifier=1 \
    --build --detach
```


A Makefile has been provided for convience. The following targets are available.
- make up
- make down
- make up-metrics
- make down-metrics

## Authentication

Influxdb has authentication disabled.

Grafana requires a login. The defaults are:
```
user: admin
password: optimism
```

## Data persistance

Grafana data is not currently saved. Any modifications or additions will be lost on container restart.

InfluxDB is persisting data to a Docker volume.

**Stopping the project removing the containers will not clear this volume**

To remove the influxdb and grafana data, run a commands like
```
docker volume rm ops_influxdb_data
docker volume rm ops_grafana_data
```

## Accessing Grafana dashboards

After starting up the project Grafana should be listening on http://localhost:3000.

Access this link and authenticate as `admin` (see #Authentication)

From the Dashboard list, select "Geth dashboard".
