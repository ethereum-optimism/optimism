---
layout:
  title:
    visible: true
  description:
    visible: false
  tableOfContents:
    visible: true
  outline:
    visible: true
  pagination:
    visible: true
---

# Welcome to Boba

This is the primary place where [Boba](https://boba.network) works on the Boba L2, a compute-focused L2. Fundamentally, Ethereum is a distributed computer. We believe that L2s can play a unique role in augmenting the base _compute_ capabilities of the Ethereum ecosystem. You can learn more about Turing hybrid compute [here](broken-reference).

<figure><img src=".gitbook/assets/Hybrid Compute page - Technical details chart.png" alt=""><figcaption></figcaption></figure>

Boba is built on the Optimistic Rollup developed by [Optimism](https://optimism.io). Aside from it focus on augmenting compute, Boba differs from Optimism by:

* providing additional cross-chain messaging such as a `message-relayer-fast`
* using different gas pricing logic
* providing a swap-based system for rapid L2->L1 exits (without the 7 day delay)
* providing a community fraud-detector that allows transactions to be independently verified by anyone
* interacting with L2 ETH using the normal ETH methods (`msg.value`, `send eth_sendTransaction`, and `provider.getBalance(address)` rather than as WETH
* being organized as a [DAO](packages/boba/contracts/contracts/DAO/)
* native [NFT bridging](broken-reference)
* automatically relaying classical 7-day exit messages to L1 for you, rather than this being a separate step



<figure><img src=".gitbook/assets/documentation.png" alt=""><figcaption></figcaption></figure>

User focused documentation is available [on the Boba docs website](http://docs.boba.network/). Developer-focused documentation lives in [`./boba_documentation`](https://github.com/bobanetwork/boba/blob/develop/boba\_documentation) and within the service and contract directories. If you have questions or feel like something is missing check out our [Discord server](https://discord.com/invite/YFweUKCb8a) where we are actively responding, or [open an issue](https://github.com/bobanetwork/boba/issues) in the GitHub repo for this site.

### Direct Support

[Telegram for Developers](https://t.me/bobadev)\
[Project Telegram](https://t.me/bobanetwork)\
[Discord](https://discord.com/invite/YFweUKCb8a)



<figure><img src=".gitbook/assets/directory-structure.png" alt=""><figcaption></figcaption></figure>

**Base Layer (generally similar to Optimistic Ethereum)**

* [`packages`](packages/): Contains all the typescript packages and contracts
  * [`contracts`](packages/contracts/): Solidity smart contracts implementing the OVM
  * [`core-utils`](packages/core-utils/): Low-level utilities and encoding packages
  * [`common-ts`](packages/common-ts/): Common tools for TypeScript code that runs in Node
  * [`data-transport-layer`](packages/data-transport-layer/): Event indexer, allowing the `l2geth` node to access L1 data
  * [`batch-submitter`](go/batch-submitter/): Daemon for submitting L2 transaction and state root batches to L1
  * [`message-relayer`](packages/message-relayer/): Service for relaying L2 messages to L1
  * [`replica-healthcheck`](packages/replica-healthcheck/): Service to monitor the health of different replica deployments
* [`l2geth`](l2geth/): Fork of [go-ethereum v1.9.10](https://github.com/ethereum/go-ethereum/tree/v1.9.10) implementing the [OVM](https://research.paradigm.xyz/optimism#optimistic-geth).
* [`integration-tests`](integration-tests/): Integration tests between a L1 testnet and the `l2geth`
* [`ops`](ops/): Contains Dockerfiles for containerizing each service involved in the protocol, as well as a docker-compose file for bringing up local testnets easily

**Boba Layer**

* [`packages/boba/turing`](broken-reference): System for hybrid compute
* [`boba_community`](boba\_community/): Code for running your own Boba node/replica and the fraud detector
* [`boba_documentation`](boba\_documentation/): Boba-specific documentation
* [`boba_examples`](boba\_examples/): Basic examples of deploying contracts on Boba
* [`boba_utilities`](boba\_utilities/): A stress-tester for discovering bugs under load
* [`ops_boba`](ops\_boba/): Parts of the Boba back-end, including the `api-watcher` service
* [`packages/boba`](packages/boba/): Contains all the Boba typescript packages and contracts
  * [`contracts`](packages/boba/contracts/): Solidity smart contracts implementing the fast bridges, the DAO, etc.
  * [`gas-price-oracle`](oracles/gas-price-oracle.md): A custom gas price oracle
  * [`gateway`](packages/boba/gateway/): The Boba Web gateway
  * [`message-relayer-fast`](packages/message-relayer/): A batch message relayer that can be run for the fast mode without a 7 day delay
  * [`register`](for-developers/register.md): Code for registering addresses in the AddressManager
  * [`subgraph`](for-developers/features/subgraph.md): Subgraphs for indexing the **StandardBridge** and **LiquidityPool** contracts



<figure><img src=".gitbook/assets/contributing.png" alt=""><figcaption></figcaption></figure>

Follow these instructions to set up your local development environment.

### Dependencies

You'll need the following:

* [Git](https://git-scm.com/downloads)
* [NodeJS](https://nodejs.org/en/download/)
* [Yarn](https://classic.yarnpkg.com/en/docs/install)
* [Docker](https://docs.docker.com/get-docker/)
* [Docker Compose](https://docs.docker.com/compose/install/)

**Note: this is only relevant to developers who wish to work on Boba core services. For most test uses, e.g. deploying your contracts, it's simpler to use https://rinkeby.boba.network**.

Clone the repository, open it, and install nodejs packages with `yarn`:

```bash
$ git clone git@github.com:bobanetwork/boba.git
$ cd boba
$ yarn clean # only needed / will only work if you had it installed previously
$ yarn
$ yarn build
```

Then, make sure you have Docker installed _and make sure Docker is running_. Finally, build and run the entire stack:

```bash
$ cd ops
$ BUILD=1 DAEMON=0 ./up_local.sh
```



<figure><img src=".gitbook/assets/spinning up the stack (1).png" alt=""><figcaption></figcaption></figure>

Stack spinup can take 15 minutes or more. There are many interdependent services to bring up with two waves of contract deployment and initialization. Recommended settings in docker - 10 CPUs, 30 to 40 GB of memory. You can either inspect the Docker `Dashboard>Containers/All>Ops` for the progress of the `ops_deployer` _or_ you can run this script to wait for the sequencer to be fully up:

```bash
./scripts/wait-for-sequencer.sh
```

If the command returns with no log output, the sequencer is up. Once the sequencer is up, you can inspect the Docker `Dashboard>Containers/All>Ops` for the progress of `ops_boba_deployer` _or_ you can run the following script to wait for all the Boba contracts (e.g. the fast message relay system) to be deployed and up:

```bash
./scripts/wait-for-boba.sh
```

When the command returns with `Pass: Found L2 Liquidity Pool contract address`, the entire Boba stack has come up correctly.



<figure><img src=".gitbook/assets/helpful commands.png" alt=""><figcaption></figcaption></figure>

* _Running out of space on your Docker, or having other having hard to debug issues_? Try running `docker system prune -a --volumes` and then rebuild the images.
* _To (re)build individual base services_: `docker-compose build -- l2geth`
* _To (re)build individual Boba ts services_: `docker-compose build -- builder` then `docker-compose build -- dtl`, for example



<figure><img src=".gitbook/assets/running unit tests (1).png" alt=""><figcaption></figcaption></figure>

To run unit tests for a specific package:

```bash
cd packages/package-to-test
yarn test
```



<figure><img src=".gitbook/assets/running integration tests (1).png" alt=""><figcaption></figcaption></figure>

Make sure you are in the `ops` folder and then run

```bash
docker-compose run integration_tests
```

Expect the full test suite with more than 110 tests including load tests to complete in between _30 minutes_ to _two hours_ depending on your computer hardware.



<figure><img src=".gitbook/assets/viewing docker container logs.png" alt=""><figcaption></figcaption></figure>

By default, the `docker-compose up` command will show logs from all services, and that can be hard to filter through. In order to view the logs from a specific service, you can run:

```bash
docker-compose logs --follow <service name>
```



<figure><img src=".gitbook/assets/license.png" alt=""><figcaption></figcaption></figure>

Code forked from [`go-ethereum`](https://github.com/ethereum/go-ethereum) under the name [`l2geth`](https://github.com/ethereum-optimism/optimism/tree/master/l2geth) is licensed under the [GNU GPLv3](https://gist.github.com/kn9ts/cbe95340d29fc1aaeaa5dd5c059d2e60) in accordance with the [original license](https://github.com/ethereum/go-ethereum/blob/master/COPYING).

All other files within this repository are licensed under the [MIT License](https://github.com/bobanetwork/boba/blob/develop/LICENSE) unless stated otherwise.
