
# immutability-eth-plugin

## Documentation

https://omisego.immutability.io

## Development Approach

This repo assumes that development and testing will be done in docker.  We use docker to:

* Build an Alpine based Vault Docker image using musl with the plugin packaged.
* Run and test the plugin using docker-compose

## Building

### Build DockerFile

`make docker-build`

Creates a local docker build of HashiCorp Vault with this plugin installed.  See the makefile for details.

### Run docker-compose

`make run`

Spins up a stateless Vault server with this plugin for testing purposes.  Once stopped all state is lost. Stdout goes to console window where the docker-compose up command was run. Ctrl-c kills the server.

See the makefile for details.

Not for production use.

### Build and Run

`make all`

### Development certs

To simplify things `make run` will generate certs in the project folder ./docker/ca. These are not meant to be used in production.

