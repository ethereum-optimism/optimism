# <h1 align="center"> Optimism Monorepo </h1>

**Monorepo implementing the Optimistic Ethereum protocol**

![Github Actions](https://github.com/ethereum-optimism/optimism/workflows/typescript%20/%20contracts/badge.svg)
![Github Actions](https://github.com/ethereum-optimism/optimism/workflows/integration/badge.svg)
![Github Actions](https://github.com/ethereum-optimism/optimism/workflows/geth%20unit%20tests/badge.svg)

## Documentation

Extensive documentation is available [here](http://community.optimism.io/docs/)

## Directory Structure

* [`packages`](./packages): Contains all the typescript packages and contracts
    * [`contracts`](./packages/contracts): Solidity smart contracts implementing the OVM
    * [`core-utils`](./packages/core-utils): Low-level utilities and encoding packages
    * [`hardhat-ovm`](./packages/hardhat-ovm): Hardhat plugin which enables the [OVM Compiler](https://github.com/ethereum-optimism/solidity)
    * [`smock`](./packages/smock): Testing utility for mocking smart contract return values and storage
    * [`data-transport-layer`](./packages/data-transport-layer): Event indexer, allowing the `l2geth` node to access L1 data
    * [`batch-submitter`](./packages/batch-submitter): Daemon for submitting L2 transaction and state root batches to L1
    * [`message-relayer`](./packages/message-relayer): Service for relaying L2 messages to L1
* [`l2geth`](./l2geth): Fork of [go-ethereum v1.9.10](https://github.com/ethereum/go-ethereum/tree/v1.9.10) implementing the [OVM](https://research.paradigm.xyz/optimism#optimistic-geth).
* [`integration-tests`](./integration-tests): Integration tests between a L1 testnet, `l2geth`,
* [`ops`](./ops): Contains Dockerfiles for containerizing each service involved in the protocol, 
as well as a docker-compose file for bringing up local testnets easily

## Quickstart

### Installation

Dependency management is done using `yarn`. 

```bash
git clone git@github.com:ethereum-optimism/optimism.git
cd optimism
yarn
```

After installing the dependencies, you must also build them so that the typescript
is compiled down to javascript:

```bash
yarn build
```

### Unit tests

All tests are run in parallel using `lerna`:

```bash
yarn test
```

When you want to run tests only for packages that have changed since `master` (or any other branch)
you can run `yarn lerna run test --parallel --since master`

### Integration Tests

The integration tests first require bringing up the Optimism stack. This is done via
a Docker Compose network. For better performance, we also recommend enabling Docker 
BuildKit

```bash
cd ops
export COMPOSE_DOCKER_CLI_BUILD=1
export DOCKER_BUILDKIT=1
docker-compose build
docker-compose up
```
