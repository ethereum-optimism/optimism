# Rollup Boost

Rollup Boost is a block builder sidecar for Optimism Stack chains to enable external block production. To read more about the design, check out the [design doc](https://github.com/ethereum-optimism/design-docs/blob/main/protocol/external-block-production.md).

## Usage

Run the rollup-boost server using the following command:

```
cargo run -- [OPTIONS]
```

### Command-line Options

- `--l2-jwt-token <TOKEN>`: JWT token for L2 authentication (required)
- `--l2-jwt-path <PATH>`: Path to the L2 JWT secret file (required if `--l2-jwt-token` is not provided)
- `--l2-url <URL>`: URL of the local L2 execution engine (required)
- `--builder-url <URL>`: URL of the builder execution engine (required)
- `--builder-jwt-token <TOKEN>`: JWT token for builder authentication (required)
- `--builder-jwt-path <PATH>`: Path to the builder JWT secret file (required if `--builder-jwt-token` is not provided)
- `--rpc-host <HOST>`: Host to run the server on (default: 0.0.0.0)
- `--rpc-port <PORT>`: Port to run the server on (default: 8081)
- `--tracing`: Enable tracing (default: false)
- `--log-level <LEVEL>`: Log level (default: info)
- `--log-format <FORMAT>`: Log format (default: text)
- `--metrics`: Enable metrics (default: false)
- `--no-boost-sync`: Disables using the proposer to sync the builder node (default: true)
- `--debug-server-port <PORT>`: Port to run the debug server on (default: 5555)

### Environment Variables

You can also set the options using environment variables. See .env.example to use the default values.

### Example

```
cargo run --l2-jwt-token your_jwt_token --l2-url http://localhost:8545 --builder-jwt-token your_jwt_token --builder-url http://localhost:8546
```

## Core System Workflow

1. `rollup-boost` receives an `engine_FCU` with the attributes to initiate block building:
   - It relays the call to proposer `op-geth` as usual and multiplexes the call to builder.
   - The FCU call returns the proposer payload id and internally maps the builder payload id to proposer payload id in the case the payload ids are not the same.
2. When `rollup-boost` receives an `engine_getPayload`:
   - It queries proposer `op-geth` for a fallback block.
   - In parallel, it queries builder for a block.
3. Upon receiving the builder block:
   - `rollup-boost` validates the block with proposer `op-geth` using `engine_newPayload`.
   - This validation ensures the block will be valid for proposer `op-geth`, preventing network stalls due to invalid blocks.
   - If the external block is valid, it is returned to the proposer `op-node`. Otherwise, `rollup-boost` will return the fallback block.
4. The proposer `op-node` sends a `engine_newPayload` request to `rollup-boost` and another `engine_FCU` without attributes to update chain state.
   - `rollup-boost` just relays the calls to proposer `op-geth`.
   - Note that since we already called `engine_newPayload` on the proposer `op-geth` in the previous step, the block should be cached and add minimal latency.
   - The builder `op-node` will receive blocks via p2p gossip and keep the builder node in sync via the engine api.

```mermaid
sequenceDiagram
    box Proposer
        participant op-node
        participant rollup-boost
        participant op-geth
    end
    box Builder
        participant builder-op-node as op-node
        participant builder-op-geth as builder
    end

    Note over op-node, builder-op-geth: 1. Triggering Block Building
    op-node->>rollup-boost: engine_FCU (with attrs)
    rollup-boost->>op-geth: engine_FCU (with attrs)
    rollup-boost->>builder-op-geth: engine_FCU (with attrs)
    rollup-boost->>op-node: proposer payload id

    Note over op-node, builder-op-geth: 2. Get Local and Builder Blocks
    op-node->>rollup-boost: engine_getPayload
    rollup-boost->>op-geth: engine_getPayload
    rollup-boost->>builder-op-geth: engine_getPayload

    Note over op-node, builder-op-geth: 3. Validating and Returning Builder Block
    rollup-boost->>op-geth: engine_newPayload
    op-geth->>rollup-boost: block validity
    rollup-boost->>op-node: block payload

    Note over op-node, builder-op-geth: 4. Updating Chain State
    op-node->>rollup-boost: engine_newPayload
    rollup-boost->>op-geth: engine_newPayload
    op-node->>rollup-boost: engine_FCU (without attrs)
    rollup-boost->>op-geth: engine_FCU (without attrs)
```

## RPC Calls

By default, `rollup-boost` will proxy all RPC calls from the proposer `op-node` to its local `op-geth` node. These are the list of RPC calls that are proxied to both the proposer and the builder execution engines:

- `engine_forkchoiceUpdatedV3`: this call is only multiplexed to the builder if the call contains payload attributes and the no_tx_pool attribute is false.
- `engine_getPayloadV3`: this is used to get the builder block.
- `miner_*`: this allows the builder to be aware of changes in effective gas price, extra data, and [DA throttling requests](https://docs.optimism.io/builders/chain-operators/configuration/batcher) from the batcher.
- `eth_sendRawTransaction*`: this forwards transactions the proposer receives to the builder for block building. This call may not come from the proposer `op-node`, but directly from the rollup's rpc engine.

### Boost Sync

By default, `rollup-boost` will sync the builder with the proposer `op-node`. After the builder is synced, boost sync improves the performance of keeping the builder in sync with the tip of the chainby removing the need to receive chain updates via p2p via the builder `op-node`. This entails additional engine api calls that are multiplexed to the builder from rollup-boost:

- `engine_forkchoiceUpdatedV3`: this call will be multiplexed to the builder regardless of whether the call contains payload attributes or not.
- `engine_newPayloadV3`: ensures the builder has the latest block if the local payload was used.

## Debug Mode

`rollup-boost` also includes a debug command to run rollup-boost with a dry run mode. This will stop get payload requests to the builder and always use the local payload.

This is useful for testing interactions with external block builders in a production environment without jeopardizing OP stack liveness, especially for network upgrades.

### Usage

To run rollup-boost in debug mode with dry run enabled, you can use the following command:

```
cargo run -- debug --dry-run
```

### Debug API

Dry run mode can also be configured via the debug api. This allows rollup-boost to stop requesting builder payloads during runtime without needing a restart. By default, the debug server runs on port 5555.

To toggle dry run mode:

```bash
curl -X POST -H "Content-Type: application/json" --data '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "debug_setDryRun",
    "params": [{"action":"toggle_dry_run"}]
}' http://localhost:5555
```

To set the dry run state:

```bash
curl -X POST -H "Content-Type: application/json" --data '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "debug_setDryRun",
    "params": [{"action":{"set_dry_run":true}}]
}' http://localhost:5555
```

## License

The code in this project is free software under the [MIT License](/LICENSE).

---

Made with ☀️ by the ⚡🤖 collective.
