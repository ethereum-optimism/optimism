# Running Rollup Boost

Rollup boost acts as a proxy between the proposer its execution engine. It is stateless and can be run with a regular sequencer setup or in a high availability setup with `op-conductor`.

![rollup-boost-architecture](../assets/rollup-boost-architecture.png)

## Regular Sequencer Setup

To run rollup boost with a regular sequencer setup, change the `--l2` flag in the proposer `op-node` to point to the rollup boost rpc endpoint.

To configure rollup-boost, set the l2 url to the url of the proposer auth rpc endpoint and the builder url to the builder auth rpc endpoint.

```bash
cargo run -- --l2-url http://localhost:8551 --builder-url http://localhost:8546
```

To set up a builder, you can use [`op-rbuilder`](https://github.com/flashbots/rbuilder/tree/develop/crates/op-rbuilder) with an op-node instance and have rollup-boost point to the builder auth rpc endpoint. It is recommended that boost sync is enabled on rollup-boost to sync the builder with the proposer op-node to removed the p2p networking overhead. In testing, this reduces latency significantly from around 200-300 milliseconds to just 3-4 milliseconds in local environments.

Depending on the block time of the rollup, you can set the `builder_timeout` flag for failsafe guarantees such that rollup-boost will not wait too long for a builder to respond. The default timeout is 200ms, with the assumption that the builder will be geographically close to the proposer. There is also a `l2_timeout` flag which is set to 2000ms by default, which is the timeout for the local execution engine to respond to the proposer.

Optionally, you can set a separate builder jwt token or path to the proposer jwt token via the `--builder-jwt-token` and `--builder-jwt-path` flags. If not provided, the proposer jwt token will be used.

## High Availability Setup with Rollup Boost

One potential way to setup rollup boost in a high availability setup is with `op-conductor` to run multiple instances of rollup boost and have them proxy to the same builder.

While this does not ensure high availability for the builder, the chain will have a high availability setup for the fallback node. If the proposer execution engine is healthy, `op-conductor` will assume the sequencer is healthy.

![rollup-boost-op-conductor](../assets/rollup-boost-op-conductor.png)

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

To check that rollup-boost is sending requests to get blocks from the builder, you can check the `builder_get_payload_v3` metric which is incremented when a `engine_getPayloadV3` call is proxied to the builder.

Additionally, execution engines such as op-rbuilder has rpc metrics exposed to check if `engine_getPayloadV3` requests have been received. To check if the builder blocks are landing on-chain, the builder can be configured to include a builder transaction in the block, which is captured as part of the builder metrics. To see more details about obserability in the op-builder, you can check op-rbuilder's [README](https://github.com/flashbots/rbuilder/tree/develop/crates/op-rbuilder).

### Tracing

Tracing is enabled by setting the `--tracing` flag. This will start exporting traces to the otlp endpoint specified in the `--otlp-endpoint` flag. This endpoint is set to `http://localhost:4317` by default.

Traces use the payload id to track the block building lifecycle. A distributed tracing system such as [Jaeger](https://www.jaegertracing.io/) can be used to visualize when the proposer triggers block building via `engine_forkchoiceUpdatedV3` and retrieve the block with `engine_getPayloadV3`.