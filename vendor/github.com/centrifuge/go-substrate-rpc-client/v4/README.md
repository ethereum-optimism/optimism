# Go Substrate RPC Client (GSRPC)

[![License: Apache v2.0](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![GoDoc Reference](https://godoc.org/github.com/centrifuge/go-substrate-rpc-client?status.svg)](https://godoc.org/github.com/centrifuge/go-substrate-rpc-client)
[![Build Status](https://travis-ci.com/centrifuge/go-substrate-rpc-client.svg?branch=master)](https://travis-ci.com/centrifuge/go-substrate-rpc-client)
[![codecov](https://codecov.io/gh/centrifuge/go-substrate-rpc-client/branch/master/graph/badge.svg)](https://codecov.io/gh/centrifuge/go-substrate-rpc-client)
[![Go Report Card](https://goreportcard.com/badge/github.com/centrifuge/go-substrate-rpc-client)](https://goreportcard.com/report/github.com/centrifuge/go-substrate-rpc-client)

Substrate RPC client in Go. It provides APIs and types around Polkadot and any Substrate-based chain RPC calls.
This client is modelled after [polkadot-js/api](https://github.com/polkadot-js/api).

## State

This package is feature complete, but it is relatively new and might still contain bugs. We advice to use it with caution in production. It comes without any warranties, please refer to LICENCE for details.

## Documentation & Usage Examples

Please refer to https://godoc.org/github.com/centrifuge/go-substrate-rpc-client

## Contributing

1. Install dependencies by running `make`
2. Build the project with `go build`
3. Lint `make lint` (you can use `make lint-fix` to automatically fix issues)
4. Run `make run-substrate-docker` to run the Substrate docker container

### Testing

We run our tests against a Substrate Docker image. You can choose to run
the tests within a tests-dedicated Docker container or without a container.

1. `make test-dockerized`
    Run tests within a docker container of its own against the Substrate docker container.

2. `make test`
    Run the tests locally against the Substrate docker container. Note that it expects the
    Substrate docker container to be up and running to execute the whole test suite properly.


Visit https://polkadot.js.org/apps for inspection

### Adding support for new RPC methods

After adding support for new methods, update the RPC mocks.

1. Install [mockery](https://github.com/vektra/mockery)
2. Run `go generate ./...`
