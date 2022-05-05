# rpc-proxy

This tool implements `proxyd`, an RPC request router and proxy. It does the following things:

1. Whitelists RPC methods.
2. Routes RPC methods to groups of backend services.
3. Automatically retries failed backend requests.
4. Provides metrics the measure request latency, error rates, and the like.

## Usage

Run `make proxyd` to build the binary. No additional dependencies are necessary.

To configure `proxyd` for use, you'll need to create a configuration file to define your proxy backends and routing rules.  Check out [example.config.toml](./example.config.toml) for how to do this alongside a full list of all options with commentary.

Once you have a config file, start the daemon via `proxyd <path-to-config>.toml`.

## Metrics

See `metrics.go` for a list of all available metrics.                                   

The metrics port is configurable via the `metrics.port` and `metrics.host` keys in the config.

## Adding Backend SSL Certificates in Docker

The Docker image runs on Alpine Linux. If you get SSL errors when connecting to a backend within Docker, you may need to add additional certificates to Alpine's certificate store. To do this, bind mount the certificate bundle into a file in `/usr/local/share/ca-certificates`. The `entrypoint.sh` script will then update the store with whatever is in the `ca-certificates` directory prior to starting `proxyd`.