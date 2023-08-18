## Development Quick Start

### Dependencies

You'll need the following:

* [Git](https://git-scm.com/downloads)
* [NodeJS](https://nodejs.org/en/download/)
* [Node Version Manager](https://github.com/nvm-sh/nvm)
* [pnpm](https://pnpm.io/installation)
* [Docker](https://docs.docker.com/get-docker/)
* [Docker Compose](https://docs.docker.com/compose/install/)
* [Go](https://go.dev/dl/)
* [Foundry](https://getfoundry.sh)
* [go-ethereum](https://github.com/ethereum/go-ethereum)

### Setup

Clone the repository and open it:

```bash
git clone git@github.com:ethereum-optimism/optimism.git
cd optimism
```

### Install the Correct Version of NodeJS

Install the correct node version with [nvm](https://github.com/nvm-sh/nvm)

```bash
nvm use
```

### Install node modules with pnpm

```bash
pnpm i
```

### Building the TypeScript packages

[foundry](https://github.com/foundry-rs/foundry) is used for some smart contract
development in the monorepo. It is required to build the TypeScript packages
and compile the smart contracts. Install foundry [here](https://getfoundry.sh/).

To build all of the [TypeScript packages](../../packages/), run:

```bash
pnpm clean
pnpm build
```

Packages compiled when on one branch may not be compatible with packages on a different branch.
**You should recompile all packages whenever you move from one branch to another.**
Use the above commands to recompile the packages.

### Building the rest of the system

If you want to run an Optimism node OR **if you want to run the integration tests**, you'll need to build the rest of the system.
Note that these environment variables significantly speed up build time.

```bash
cd ops-bedrock
export COMPOSE_DOCKER_CLI_BUILD=1
export DOCKER_BUILDKIT=1
docker-compose build
```

Source code changes can have an impact on more than one container.
**If you're unsure about which containers to rebuild, just rebuild them all**:

```bash
cd ops-bedrock
docker-compose down
docker-compose build
docker-compose up
```

**If a node process exits with exit code: 137** you may need to increase the default memory limit of docker containers

Finally, **if you're running into weird problems and nothing seems to be working**, run:

```bash
cd optimism
pnpm clean
pnpm build
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
pnpm test
```

To run unit tests for a specific package:

```bash
cd packages/package-to-test
pnpm test
```

#### Running contract static analysis

We perform static analysis with [`slither`](https://github.com/crytic/slither).
You must have Python 3.x installed to run `slither`.
To run `slither` locally, do:

```bash
cd packages/contracts
pip3 install slither-analyzer
pnpm test:slither
```
