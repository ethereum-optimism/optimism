- [Starting a local basic Optimism L1/L2 with OMGX contracts and services](#starting-a-local-basic-optimism-l1-l2-with-omgx-contracts-and-services)
  * [Starting a local basic Optimism L1/L2](#starting-a-local-basic-optimism-l1-l2)
    + [Overall Setup](#overall-setup)
  * [(Re)Building the entire system or parts of the base L1/L2](#-re-building-the-entire-system-or-parts-of-the-base-l1-l2)
  * [(Re)Building the entire system or parts of the OMGX contracts and services](#-re-building-the-entire-system-or-parts-of-the-omgx-contracts-and-services)
      - [Viewing docker container logs](#viewing-docker-container-logs)
    + [Running unit tests](#running-unit-tests)
    + [Running integration tests](#running-integration-tests)

# Overall Setup

Clone the repository, open it, and install nodejs packages with `yarn`:

```bash
git clone git@github.com:omgnetwork/optimism.git
cd optimism
yarn clean
yarn install
yarn build
```
With all this done we can move on to actually spinning up a local version of the Optimism L1/L2.
  
**NOTE: You should recompile all packages whenever you move from one  branch to another.**  
Use the below commands to recompile the packages.

Note: _Running out of space on your Docker, or having other having hard to debug issues_? Try running `docker system prune -a --volumes` and then rebuild the images. 

## Starting a local basic Optimism L1/L2

You can change the BUILD and DAEMON values to control if everything is rebuilt (`BUILD=1`, very slow), and if you want to see all the debug information (`DAEMON=0`)

**Before running any Docker related commands make sure you have Docker up and running.**

```bash
cd ops
export COMPOSE_DOCKER_CLI_BUILD=1 # these environment variables significantly speed up build time
export DOCKER_BUILDKIT=1
docker-compose build 
docker-compose up -V
```


## (Re)Building the entire system or parts of the base L1/L2

If you want to run an Optimistic Ethereum node OR **if you want to run the integration tests**, you'll need to build the rest of the system.

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

## (Re)Building the entire system or parts of the OMGX contracts and services

```bash
cd ops
export COMPOSE_DOCKER_CLI_BUILD=1 # these environment variables significantly speed up build time
export DOCKER_BUILDKIT=1
docker-compose build 
docker-compose -f docker-compose.yml up -V
```

To build individual OMGX services:

```bash
docker-compose build -- omgx_message-relayer-fast
```

**Note: First you will have to comment out various dependencies in the `docker-compose.yml`.**

#### Viewing docker container logs

By default, the `docker-compose up` command will show logs from all services, and that
can be hard to filter through. In order to view the logs from a specific service, you can run:

```bash
docker-compose logs --follow <service name>
```

### Running unit tests

Before running tests: **follow the above instructions to get everything built.** Run unit tests for all packages in parallel via:

```bash
yarn test
```

To run unit tests for a specific package:

```bash
cd packages/package-to-test
yarn test
```

### Running integration tests

Follow above instructions for building the whole stack. Build and run the integration tests:

```bash
cd integration-tests
yarn build:integration
yarn test:integration
```

