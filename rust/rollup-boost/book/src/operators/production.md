# Running Rollup Boost in Production

## Regular Sequencer Setup

To run rollup-boost with a regular sequencer setup, change the `--l2` flag in the proposer `op-node` to point to the rollup-boost RPC endpoint.

To configure rollup-boost, set the L2 URL to the URL of the proposer auth RPC endpoint and the builder URL to the builder auth RPC endpoint. Separate JWT tokens will be needed for the two endpoints.

You can also set the options using environment variables. See `.env.example` to use the default values.

```bash
cargo run --bin rollup-boost -- \
    --l2-jwt-token your_jwt_token \
    --l2-url http://localhost:8545 \
    --builder-jwt-token your_jwt_token \
    --builder-url http://localhost:8546
```

To set up a builder, you can use [`op-rbuilder`](https://github.com/flashbots/op-rbuilder) with an op-node instance and have rollup-boost point to the builder auth RPC endpoint.

## Flashblocks

To launch rollup-boost with Flashblocks enabled:

```bash
cargo run --bin rollup-boost -- \
  --l2-url http://localhost:5555 \
  --builder-url http://localhost:4445 \
  --l2-jwt-token 688f5d737bad920bdfb2fc2f488d6b6209eebda1dae949a8de91398d932c517a \
  --builder-jwt-token 688f5d737bad920bdfb2fc2f488d6b6209eebda1dae949a8de91398d932c517a \
  --rpc-port 4444 \
  --flashblocks \
  --log-level info
```

This command uses the default Flashblocks configuration. For custom configurations, see the [Flashblocks Configuration](#flashblocks-configuration) section below.

#### Default Port Configuration

- `4444`: rollup-boost RPC port
- `4445`: op-rbuilder auth RPC port (matches rollup-boost builder URL)
- `5555`: op-reth auth RPC port (matches rollup-boost L2 URL)
- `3030`: op-rbuilder P2P port
- `3131`: op-reth P2P port

### Flashblocks Configuration

rollup-boost provides several configuration options for Flashblocks functionality:

#### Basic Flashblocks Flag

- `--flashblocks`: Enable Flashblocks client (required)
  - Environment variable: `FLASHBLOCKS`

#### WebSocket Connection Settings

- `--flashblocks-builder-url <URL>`: Flashblocks Builder WebSocket URL

  - Environment variable: `FLASHBLOCKS_BUILDER_URL`
  - Default: `ws://127.0.0.1:1111`

- `--flashblocks-host <HOST>`: Flashblocks WebSocket host for outbound connections

  - Environment variable: `FLASHBLOCKS_HOST`
  - Default: `127.0.0.1`

- `--flashblocks-port <PORT>`: Flashblocks WebSocket port for outbound connections
  - Environment variable: `FLASHBLOCKS_PORT`
  - Default: `1112`

#### Connection Management

- `--flashblock-builder-ws-reconnect-ms <MILLISECONDS>`: Timeout duration if builder disconnects
  - Environment variable: `FLASHBLOCK_BUILDER_WS_RECONNECT_MS`
  - No default value specified

## Execution mode

`ExecutionMode` is a configuration setting that controls how `rollup-boost` interacts with the external builder during block production. Execution mode can be set either at startup via CLI flags or dynamically modified at runtime through the [Debug API](#debug-api).
Operators can use `ExecutionMode` to selectively forward or bypass builder interactions, enabling dry runs during deployments or fully disabling external block production during emergencies.

The available execution modes are:

- `Enabled`
  - `rollup-boost` forwards all Engine API requests to both the builder and default execution client.
  - Optimistically selects the builder’s payload for validation and block publication.
  - Falls back to the local execution client *only* if the builder fails to produce a payload or the payload is invalid.
  - Default setting for normal external block production.

- `DryRun`  
  - `rollup-boost` forwards all Engine API requests to both the builder and default execution client.
  - Builder payloads are validated with the local execution client but the default execution client block will always be returned to `op-node` to propagate to the network.
  - Useful during deployments, dry runs, or to validate builder behavior without publishing builder blocks to the network.

- `Disabled`
  - `rollup-boost` does not forward any Engine API requests to the builder.
  - Block construction is handled exclusively by the default execution client.
  - Useful as an emergency shutoff switch in the case of critical failures/emergencies.

```rust
pub enum ExecutionMode {
    /// Forward Engine API requests to the builder, validate builder payloads and propagate to the network 
    Enabled,
    /// Forward Engine API requests to the builder, validate builder payloads but
    /// fallback to default execution payload
    DryRun,
    // Do not forward Engine API requests to the builder 
    Disabled,
}
```

<br>

## Debug API

`rollup-boost` exposes a Debug API that allows operators to inspect and modify the current execution mode at runtime without restarting the service. This provides flexibility to dynamically enable, disable, or dry-run external block production based on builder behavior or network conditions. The Debug API is served over HTTP using JSON RPC and consists of the following endpoints:

### `debug_setExecutionMode`

Sets the current execution mode for `rollup-boost`.

**Request**:

```json
{
  "method": "debug_setExecutionMode",
  "params": [ "enabled" | "dry_run" | "disabled" ],
  "id": 1,
  "jsonrpc": "2.0"
}
```

**Response**:

```json
{
  "result": null,
  "id": 1,
  "jsonrpc": "2.0"
}
```

### `debug_getExecutionMode`

Retrieves the current execution mode.

**Request**:

```json
{
  "method": "debug_getExecutionMode",
  "params": [],
  "id": 1,
  "jsonrpc": "2.0"
}
```

**Response:**

```json
{
  "result": "enabled" | "dry_run" | "disabled",
  "id": 1,
  "jsonrpc": "2.0"
}
```

## Observability

### Metrics

To enable metrics, you can set the `--metrics` flag. This will start a metrics server which will run on port 9090 by default. To see the list of metrics, you can check out metrics.rs and ping the metrics endpoint:

```bash
curl http://localhost:9090/metrics
```

All spans create duration histogram metrics with the name "{span_name}\_duration". Currently, this list includes:

- fork_choice_updated_duration
- get_payload_duration
- new_payload_duration

Additionally, execution engines such as op-rbuilder have RPC metrics exposed to check if `engine_getPayloadV4` requests have been received. To check if the builder blocks are landing on-chain, the builder can be configured to include a builder transaction in the block, which is captured as part of the builder metrics. To see more details about observability in the op-builder, you can check op-rbuilder's [README](https://github.com/flashbots/op-rbuilder?tab=readme-ov-file#observability).

#### Flashblocks

There are metrics and observability in all the services supporting Flashblocks. In rollup-boost:

- `messages_processed` - number of messages processed from the Flashblocks WebSocket stream
- `flashblocks_counter` - number of Flashblocks proposed
- `flashblocks_missing_counter` - number of Flashblocks missed from the expected number of Flashblocks

Additionally, the builder transaction can also be observed in the last Flashblock to determine if the number of expected Flashblocks has been included on-chain.

### Tracing

Tracing is enabled by setting the `--tracing` flag. This will start exporting traces to the otlp endpoint specified in the `--otlp-endpoint` flag. This endpoint is set to `http://localhost:4317` by default.

Traces use the payload id to track the block building lifecycle. A distributed tracing system such as [Jaeger](https://www.jaegertracing.io/) can be used to visualize when the proposer triggers block building via `engine_forkchoiceUpdated` and retrieve the block with `engine_getPayload`.

## Troubleshooting Builder Responses

### Invalid Builder Payloads

If there are logs around the builder payload being invalid, it is likely there is an issue with the builder and you will need to contact the builder operator to resolve it. In this case rollup-boost will use the local payload and chain liveness will not be affected. You can also manually set rollup-boost to dry run mode using the debug api to stop payload requests to the builder, silencing the error logs.

It is also possible that either the builder or the proposer execution engine are not running on compatible hard fork versions. Please check that the clients are running on compatible versions of the op-stack.

### Builder Syncing

Alternatively, the builder may be syncing with the chain and not have a block to respond with. You can see in the logs the builder is syncing by checking whether the payload_status of builder calls is `SYNCING`.

This is expected if the builder is still syncing with the chain. Chain liveness will not be affected as rollup-boost will use the local payload. Contact the builder operator if the sync status persists as the builder op-node may be offline or not peered correctly with the network.