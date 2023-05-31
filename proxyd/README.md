# rpc-proxy

This tool implements `proxyd`, an RPC request router and proxy. It does the following things:

1. Whitelists RPC methods.
2. Routes RPC methods to groups of backend services.
3. Automatically retries failed backend requests.
4. Track backend consensus (`latest`, `safe`, `finalized` blocks), peer count and sync state.
5. Re-write requests and responses to enforce consensus.
6. Load balance requests across backend services.
7. Cache immutable responses from backends.
8. Provides metrics the measure request latency, error rates, and the like.


## Usage

Run `make proxyd` to build the binary. No additional dependencies are necessary.

To configure `proxyd` for use, you'll need to create a configuration file to define your proxy backends and routing rules.  Check out [example.config.toml](./example.config.toml) for how to do this alongside a full list of all options with commentary.

Once you have a config file, start the daemon via `proxyd <path-to-config>.toml`.


## Consensus awareness

Starting on v4.0.0, `proxyd` is aware of the consensus state of its backends. This helps minimize chain reorgs experienced by clients.

To enable this behavior, you must set `consensus_aware` value to `true` in the backend group.

When consensus awareness is enabled, `proxyd` will poll the backends for their states and resolve a consensus group based on:
* the common ancestor `latest` block, i.e. if a backend is experiencing a fork, the fork won't be visible to the clients
* the lowest `safe` block
* the lowest `finalized` block
* peer count
* sync state

The backend group then acts as a round-robin load balancer distributing traffic equally across healthy backends in the consensus group, increasing the availability of the proxy.

A backend is considered healthy if it meets the following criteria:
* not banned
* avg 1-min moving window error rate ≤ configurable threshold
* avg 1-min moving window latency ≤ configurable threshold
* peer count ≥ configurable threshold
* `latest` block lag ≤ configurable threshold
* last state update ≤ configurable threshold
* not currently syncing

When a backend is experiencing inconsistent consensus, high error rates or high latency,
the backend will be banned for a configurable amount of time (default 5 minutes)
and won't receive any traffic during this period.


## Tag rewrite

When consensus awareness is enabled, `proxyd` will enforce the consensus state transparently for all the clients.

For example, if a client requests the `eth_getBlockByNumber` method with the `latest` tag,
`proxyd` will rewrite the request to use the resolved latest block from the consensus group
and forward it to the backend.

The following request methods are rewritten:
* `eth_getLogs`
* `eth_newFilter`
* `eth_getBalance`
* `eth_getCode`
* `eth_getTransactionCount`
* `eth_call`
* `eth_getStorageAt`
* `eth_getBlockTransactionCountByNumber`
* `eth_getUncleCountByBlockNumber`
* `eth_getBlockByNumber`
* `eth_getTransactionByBlockNumberAndIndex`
* `eth_getUncleByBlockNumberAndIndex`
* `debug_getRawReceipts`

And `eth_blockNumber` response is overridden with current block consensus.


## Cacheable methods

Cache use Redis and can be enabled for the following immutable methods:

* `eth_chainId`
* `net_version`
* `eth_getBlockTransactionCountByHash`
* `eth_getUncleCountByBlockHash`
* `eth_getBlockByHash`
* `eth_getTransactionByBlockHashAndIndex`
* `eth_getUncleByBlockHashAndIndex`
* `debug_getRawReceipts` (block hash only)

## Metrics

See `metrics.go` for a list of all available metrics.

The metrics port is configurable via the `metrics.port` and `metrics.host` keys in the config.

## Adding Backend SSL Certificates in Docker

The Docker image runs on Alpine Linux. If you get SSL errors when connecting to a backend within Docker, you may need to add additional certificates to Alpine's certificate store. To do this, bind mount the certificate bundle into a file in `/usr/local/share/ca-certificates`. The `entrypoint.sh` script will then update the store with whatever is in the `ca-certificates` directory prior to starting `proxyd`.
