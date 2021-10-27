# rpc-proxy

This tool implements `proxyd`, an RPC request router and proxy. It does the following things:

1. Whitelists RPC methods.
2. Routes RPC methods to groups of backend services.
3. Automatically retries failed backend requests.
4. Provides metrics the measure request latency, error rates, and the like.

## Usage

Run `make proxyd` to build the binary. No additional dependencies are necessary.

To configure `proxyd` for use, you'll need to create a configuration file to define your proxy backends and routing rules. An example config that routes `eth_chainId` between Infura and Alchemy is below:

```toml
[backends]
[backends.infura]
base_url = "url-here"

[backends.alchemy]
base_url = "url-here"

[backend_groups]
[backend_groups.main]
backends = ["infura", "alchemy"]

[method_mappings]
eth_chainId = "main"
```

Check out [example.config.toml](./example.config.toml) for a full list of all options with commentary.

Once you have a config file, start the daemon via `proxyd <path-to-config>.toml`.

## Metrics

The following Prometheus metrics are exported:

| Name                                           | Description                                                                                     | Flags                                  |
|------------------------------------------------|-------------------------------------------------------------------------------------------------|----------------------------------------|
| `proxyd_backend_requests_total`                  | Count of all successful requests to a backend.                                                  | backend_name: The name of the backend. |
| `proxyd_backend_errors_total`                    | Count of all backend errors.                                                                    | backend_name: The name of the backend  |
| `proxyd_http_requests_total`                     | Count of all HTTP requests, successful or not.                                                  |                                        |
| `proxyd_http_request_duration_histogram_seconds` | Histogram of HTTP request durations.                                                            |                                        |
| `proxyd_rpc_requests_total`                      | Count of all RPC requests.                                                                      | method_name: The RPC method requested. |
| `proxyd_blocked_rpc_requests_total`              | Count of all RPC requests with a blacklisted method.                                            | method_name: The RPC method requested. |
| `proxyd_rpc_errors_total`                        | Count of all RPC errors. **NOTE:** Does not include errors sent from the backend to the client. |                                   

The metrics port is configurable via the `metrics.port` and `metrics.host` keys in the config.

## Errata

- RPC errors originating from the backend (e.g., any backend response containing  an `error` key) are passed on to the client directly. This simplifies the code and avoids having to marshal/unmarshal the backend's response JSON.
- Requests are distributed round-robin between backends in a group.