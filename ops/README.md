# docker-compose

The docker-compose project runs a local optimism stack.

## prerequisites

- docker
- docker-compose
- make

## Building the services

```bash
make build
```

## Starting and stopping the project

The base `docker-compose.yml` file will start the required components for a full stack.

Supplementing the base configuration is an additional metric enabling file, `docker-compose-metrics.yml`. Adding this configuration to the stack will enable metric emission for l2geth and start grafana (for metrics visualisation) and influxdb (for metric collection) instances.

Also available for testing is the `rpc-proxy` service in the `docker-compose-rpc-proxy.yml` file. It can be used to restrict what RPC methods are allowed to the Sequencer.

The base stack can be started and stopped with a command like this:
```
docker-compose \
    -f docker-compose.yml \
    up --build --detach
```

*Note*: This generates a large amount of log data which docker stores by default. See [Disk Usage](#disk-usage).

Also note that Docker Desktop only allocates 2GB of memory by default, which isn't enough to run the docker-compose services reliably.

To allocate more memory, go to Settings > Resources in the Docker UI and use the slider to change the value (_8GB recommended_). Make sure to click Apply & Restart for the changes to take effect.

To start the stack with monitoring enabled, just add the metric composition file.
```
docker-compose \
    -f docker-compose.yml \
    -f docker-compose-metrics.yml \
    up --build --detach
```

Optionally, run a verifier along the rest of the stack. Run a replica with the same command by switching the service name!

```
docker-compose
    -f docker-compose.yml \
    up --scale \
    verifier=1 \
    --build --detach
```

A Makefile has been provided for convience. The following targets are available.
- make up
- make down
- make up-metrics
- make down-metrics

## Turning off L2 Fee Enforcement

Fees can be turned off at runtime by setting the environment variable
`ROLLUP_ENFORCE_FEES` to `false`.

```bash
ROLLUP_ENFORCE_FEES=false docker-compose up
```

## Cross domain communication

By default, the `message-relayer` service is turned off. This means that
any tests must manually submit withdrawals. The `message-relayer` will
automatically look for withdrawals and submit the proofs. To run with the
`message-relayer` on, use the command:

```bash
$ docker-compose up --scale relayer=1
```

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

## Disk Usage

The logs generated are in the gigabytes per day range, so you need to be wary of disk exhaustion when running for long periods.

One way to solve this is to configure `/etc/docker/daemon.json` like this:

```json
{
  "log-driver": "json-file",
  "log-opts": {
    "max-size": "10m",
    "max-file": "3"
  }
}
```

This configures log rotation with a limit of 10MB per file and storing a maximum of three files (per container). [More details on docker logging configuration](https://docs.docker.com/config/containers/logging/configure/).

You can also decrease logging by increasing polling intervals:

```env
DATA_TRANSPORT_LAYER__POLLING_INTERVAL=100
```
- [./envs/dtl.env#L7](./envs/dtl.env#L7)

```env
ROLLUP_POLL_INTERVAL_FLAG=500ms
```
- [./envs/geth.env#L8](./envs/geth.env#L8)
