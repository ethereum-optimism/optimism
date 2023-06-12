# @eth-optimism/indexer

## Getting started

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

