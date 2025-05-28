# Flashblocks Websocket Proxy

## Overview
The Flashblocks Websocket Proxy is a service that subscribes to new Flashblocks from
[rollup-boost](https://github.com/flashbots/rollup-boost) on the sequencer. Then broadcasts them out to any downstream 
RPC nodes. Minimizing the number of connections to the sequencer and restricting access.

> ⚠️ **Warning**
> 
> This is currently alpha software -- deploy at your own risk!
> 
> Currently, this project is a one-directional generic websocket proxy. It doesn't inspect any data or validate clients.
> This may not always be the case.

## For Developers

### Contributing

### Building & Testing
You can build and test the project using [Cargo](https://doc.rust-lang.org/cargo/). Some useful commands are:
```
# Build the project
cargo build

# Run all the tests (requires local version of redis to be installed)
cargo test --all-features
```

### Deployment
Builds of the websocket proxy [are provided](https://github.com/base/flashblocks-websocket-proxy/pkgs/container/flashblocks-websocket-proxy).
The only configuration required is the rollup-boost URL to proxy. You can set this via an env var `UPSTREAM_WS` or a flag `--upstream-ws`.


You can see a full list of parameters by running:

`docker run ghcr.io/base/flashblocks-websocket-proxy:master --help`

### Redis Integration

The proxy supports distributed rate limiting with Redis. This is useful when running multiple instances of the proxy behind a load balancer, as it allows rate limits to be enforced across all instances.

To enable Redis integration, use the following parameters:

- `--redis-url` - Redis connection URL (e.g., `redis://localhost:6379`)
- `--redis-key-prefix` - Prefix for Redis keys (default: `flashblocks`)

Example:

```bash
docker run ghcr.io/base/flashblocks-websocket-proxy:master \
  --upstream-ws wss://your-sequencer-endpoint \
  --redis-url redis://redis:6379 \
  --global-connections-limit 1000 \
  --per-ip-connections-limit 10
```

When Redis is enabled, the following features are available:

- Distributed rate limiting across multiple proxy instances
- Connection tracking persists even if the proxy instance restarts
- More accurate global connection limiting in multi-instance deployments

If the Redis connection fails, the proxy will automatically fall back to in-memory rate limiting.

