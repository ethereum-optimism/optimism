# Please see the main README.MD for the basics. 

## Development Quick Start

### Dependencies

You'll need the following:

* [Git](https://git-scm.com/downloads)
* [NodeJS](https://nodejs.org/en/download/)
* [Yarn](https://classic.yarnpkg.com/en/docs/install)
* [Docker](https://docs.docker.com/get-docker/)
* [Docker Compose](https://docs.docker.com/compose/install/)

### Setup

Clone the repository, open it, and install nodejs packages with `yarn`:

```bash
git clone git@github.com:omgnetwork/optimism.git
cd optimism
yarn install
```

### Building the TypeScript packages

To build all of the [TypeScript packages](./packages), run:

```bash
yarn clean
yarn build
```

Packages compiled when on one branch may not be compatible with packages on a different branch.
**You should recompile all packages whenever you move from one branch to another.**
Use the above commands to recompile the packages.

### Building the rest of the system

If you want to run an Optimistic Ethereum node OR **if you want to run the integration tests**, you'll need to build the rest of the system.

```bash
cd ops
export COMPOSE_DOCKER_CLI_BUILD=1 # these environment variables significantly speed up build time
export DOCKER_BUILDKIT=1
docker-compose build 
docker-compose -f docker-compose-omgx.yml up 
```

This will build the following containers:
* [`builder`](https://hub.docker.com/r/ethereumoptimism/builder): used to build the TypeScript packages
* [`l1_chain`](https://hub.docker.com/r/ethereumoptimism/hardhat): simulated L1 chain using hardhat-evm as a backend
* [`deployer`](https://hub.docker.com/r/ethereumoptimism/deployer): process that deploys L1 smart contracts to the L1 chain
* [`dtl`](https://hub.docker.com/r/ethereumoptimism/data-transport-layer): service that indexes transaction data from the L1 chain
* [`l2geth`](https://hub.docker.com/r/ethereumoptimism/l2geth): L2 geth node running in Sequencer mode
* [`verifier`](https://hub.docker.com/r/ethereumoptimism/go-ethereum): L2 geth node running in Verifier mode
* [`relayer`](https://hub.docker.com/r/ethereumoptimism/message-relayer): helper process that relays messages between L1 and L2
* [`batch_submitter`](https://hub.docker.com/r/ethereumoptimism/batch-submitter): service that submits batches of Sequencer transactions to the L1 chain
* [`integration_tests`](https://hub.docker.com/r/ethereumoptimism/integration-tests): integration tests in a box

If you want to make a change to a container, you'll need to take it down and rebuild it.
For example, if you make a change in l2geth:

```bash
cd ops
docker-compose stop -- l2geth
docker-compose build -- l2geth
docker-compose start l2geth
```

For the typescript services, you'll need to rebuild the `builder` so that the compiled
files are re-generated, and then your service, e.g. for the batch submitter

```bash
cd ops
docker-compose stop -- batch_submitter
docker-compose build -- builder batch_submitter
docker-compose start batch_submitter
```

Source code changes can have an impact on more than one container.
**If you're unsure about which containers to rebuild, just rebuild them all**:

```bash
cd ops
docker-compose down
docker-compose build
docker-compose up
```

Finally, **if you're running into weird problems and nothing seems to be working**, run:

```bash
cd optimism
yarn clean
yarn build
cd ops
docker-compose down -v
docker-compose build
docker-compose up
```

#### Viewing docker container logs
By default, the `docker-compose up` command will show logs from all services, and that
can be hard to filter through. In order to view the logs from a specific service, you can run:

```bash
docker-compose logs --follow <service name>
```

### Running tests

Before running tests: **follow the above instructions to get everything built.**

#### Running unit tests

Run unit tests for all packages in parallel via:

```bash
yarn test
```

To run unit tests for a specific package:

```bash
cd packages/package-to-test
yarn test
```

#### Running integration tests

Follow above instructions for building the whole stack.
Build and run the integration tests:

```bash
cd integration-tests
yarn build:integration
yarn test:integration
```
