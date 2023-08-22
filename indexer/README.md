# @eth-optimism/indexer

## Getting started


### Setup env
The `indexer.toml` stores a set of preset environmental variables that can be used to run the indexer with the exception of the network specific `l1-rpc` and `l2-rpc` variables. The `indexer.toml` file can be ran as a default config, otherwise a custom `.toml` config can provided via the `--config` flag when running the application. An optional `l1-starting-height` value can be provided to the indexer to specify the L1 starting block height to begin indexing from. This should be ideally be an L1 block that holds a correlated L2 genesis commitment. Furthermore, this value must be less than the current L1 block height to pass validation. If no starting height value is provided and the database is empty, the indexer will begin sequentially processing from L1 genesis.

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

