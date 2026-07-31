# Running Rollup Boost

Rollup boost acts as a proxy between the proposer and its execution engine. It is stateless and can be run with a regular sequencer setup or in a high availability setup with `op-conductor`.

![rollup-boost-architecture](../assets/rollup-boost-architecture.png)

## Regular Sequencer Setup

To run rollup boost with a regular sequencer setup, change the `--l2` flag in the proposer `op-node` to point to the rollup boost rpc endpoint.

To configure rollup-boost, set the l2 url to the url of the proposer auth rpc endpoint and the builder url to the builder auth rpc endpoint.

```bash
cargo run --bin rollup-boost -- --l2-url http://localhost:8551 --builder-url http://localhost:8546
```

To set up a builder, you can use [`op-rbuilder`](https://github.com/flashbots/op-rbuilder) with an op-node instance and have rollup-boost point to the builder auth rpc endpoint. It is recommended that boost sync is enabled on rollup-boost to sync the builder with the proposer op-node to remove the p2p networking overhead. In testing, this reduces latency significantly from around 200-300 milliseconds to just 3-4 milliseconds in local environments.

Depending on the block time of the rollup, you can set the `builder_timeout` flag for failsafe guarantees such that rollup-boost will not wait too long for a builder to respond. The default timeout is 200ms, with the assumption that the builder will be geographically close to the proposer. There is also a `l2_timeout` flag which is set to 2000ms by default, which is the timeout for the local execution engine to respond to the proposer.

Optionally, you can set a separate builder jwt token or path to the proposer jwt token via the `--builder-jwt-token` and `--builder-jwt-path` flags. If not provided, the proposer jwt token will be used.

## High Availability Setup with Rollup Boost

One potential way to setup rollup boost in a high availability setup is with `op-conductor` to run multiple instances of rollup boost and have them proxy to the same builder.

While this does not ensure high availability for the builder, the chain will have a high availability setup for the fallback node. If the proposer execution engine is healthy, `op-conductor` will assume the sequencer is healthy.

![rollup-boost-op-conductor](../assets/rollup-boost-op-conductor.png)

### Health Checks

`rollup-boost` supports the standard array of kubernetes probes:

- `/healthz` Returns various status codes to communicate `rollup-boost` health - 200 OK - The builder is producing blocks - 206 Partial Content - The l2 is producing blocks, but the builder is not - 503 Service Unavailable - Neither the l2 or the builder is producing blocks
  `op-conductor` should eventually be able to use this signal to switch to a different sequencer in an HA sequencer setup. In a future upgrade to `op-conductor`, A sequencer leader with a healthy (200 OK) EL (`rollup-boost` in our case) could be selected preferentially over one with an unhealthy (206 or 503) EL. If no ELs are healthy, then we can fallback to an EL which is responding with `206 Partial Content`.

- `/readyz` Used by kubernetes to determine if the service is ready to accept traffic. Should always respond with `200 OK`

- `/livez` determines wether or not `rollup-boost` is live (running and not deadlocked) and responding to requests. If `rollup-boost` fails to respond, kubernetes can use this as a signal to restart the pod. Should always respond with `200 OK`

## Observability

To check if the rollup-boost server is running, you can check the health endpoint:

```
curl http://localhost:8081/healthz
```

### Metrics

To enable metrics, you can set the `--metrics` flag. This will start a metrics server which will run on port 9090 by default. To see the list of metrics, you can checkout [metrics.rs](../src/metrics.rs) and ping the metrics endpoint:

```
curl http://localhost:9090/metrics
```

All spans create duration histogram metrics with the name "{span_name}\_duration". Currently, this list includes:

- fork_choice_updated_v3_duration
- get_payload_v3_duration
- new_payload_v3_duration

Additionally, execution engines such as op-rbuilder has rpc metrics exposed to check if `engine_getPayloadV3` requests have been received. To check if the builder blocks are landing on-chain, the builder can be configured to include a builder transaction in the block, which is captured as part of the builder metrics. To see more details about observability in the op-builder, you can check op-rbuilder's [README](https://github.com/flashbots/rollup-boost?tab=readme-ov-file#rollup-boost).

### Tracing

Tracing is enabled by setting the `--tracing` flag. This will start exporting traces to the otlp endpoint specified in the `--otlp-endpoint` flag. This endpoint is set to `http://localhost:4317` by default.

Traces use the payload id to track the block building lifecycle. A distributed tracing system such as [Jaeger](https://www.jaegertracing.io/) can be used to visualize when the proposer triggers block building via `engine_forkchoiceUpdatedV3` and retrieve the block with `engine_getPayloadV3`.

## Troubleshooting Builder Responses

### Invalid Builder Payloads

If there are logs around the builder payload being invalid, it is likely there is an issue with the builder and you will need to contact the builder operator to resolve it. In this case rollup-boost will use the local payload and chain liveness will not be affected. You can also manually set rollup-boost to dry run mode using the debug api to stop payload requests to the builder, silencing the error logs.

It is also possible that either the builder or the proposer execution engine are not running on compatible hard fork versions. Please check that the clients are running on compatible versions of the op-stack.

### Builder Syncing

Alternatively, the builder may be syncing with the chain and not have a block to respond with. You can see in the logs the builder is syncing by checking whether the payload_status of builder calls is `SYNCING`.

This is expected if the builder is still syncing with the chain. Chain liveness will not be affected as rollup-boost will use the local payload. Contact the builder operator if the sync status persists as the builder op-node may be offline or not peered correctly with the network.
