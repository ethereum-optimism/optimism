# ⚠️  Important
This project has been moved to [ethereum-optimism/infra](https://github.com/ethereum-optimism/infra)

# OP User Facing Monitoring

This project simulates a synthetic user interacting with a OP Stack chain.

It is intended to be used as a tool for monitoring
the health of the network by measuring end-to-end transaction latency.


## Metrics

* Round-trip duration time to get transaction receipt (from creation timestamp)

* First-seen duration time (from creation timestamp)


## Usage

Run `make ufm` to build the binary. No additional dependencies are necessary.

Copy `example.config.toml` to `config.toml` and edit the file to configure the service.

Start the service with `ufm config.toml`.

