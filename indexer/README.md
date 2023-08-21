# @eth-optimism/indexer

## Getting started


### Setup env
The `indexer.toml` stores a set of preset environmental variables that can be used to run the indexer with the exception of the network specific `l1-rpc` and `l2-rpc` variables. The `indexer.toml` file can be ran as a default config, otherwise a custom `.toml` config can provided via the `--config` flag when running the application. Additionally, L1 system contract addresses must provided for the specific OP Stack network actively being indexed. Currently the indexer has no way to infer L1 system config addresses provided a L2 chain ID or network enum.

### Testing
All tests can be ran by running `make test` from the `/indexer` directory.  This will run all unit and e2e tests.

**NOTE:** Successfully running the E2E tests requires spinning up a local L1 geth node and pre-populating it with necessary bedrock genesis state.  This can be done by calling `make devnet-allocs` from the root of the optimism monorepo before running the indexer tests. More information on this can be found in the [op-e2e README](../op-e2e/README.md).

### Run indexer vs goerli

- install docker
- `cp example.env .env`
- fill in .env
- run `docker-compose up` to start the indexer vs optimism goerli network

### Run indexer with go

See the flags in `flags.go` for reference of what command line flags to pass to `go run`

### Run indexer vs devnet

TODO add indexer to the optimism devnet compose file (previously removed for breaking CI)

### Run indexer vs a custom configuration

`docker-compose.dev.yml` is git ignored.   Fill in your own docker-compose file here.

