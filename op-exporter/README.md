# op_exporter

A prometheus exporter to collect information from an Optimism node and serve metrics for collection

## Usage

```
make build && ./op_exporter --rpc.provider="https://kovan-sequencer.optimism.io" --label.network="kovan"
```

## Health endpoint `/health`

Returns json describing the health of the sequencer based on the time since a block height update.

```
$ curl http://localhost:9100/health
{ "healthy": "false" }
```

## Metrics endpoint `/metrics`

```
# HELP op_gasPrice Gas price.
# TYPE op_gasPrice gauge
op_gasPrice{layer="layer1",network="kovan"} 6.9e+09
op_gasPrice{layer="layer2",network="kovan"} 1
```
