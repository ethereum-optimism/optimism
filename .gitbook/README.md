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

This is the primary place where [Boba](https://boba.network) works on the Boba L2, a compute-focused L2. Fundamentally, Ethereum is a distributed computer. We believe that L2s can play a unique role in augmenting the base _compute_ capabilities of the Ethereum ecosystem. You can learn more about Turing hybrid compute [here](contents/hc/README.md).

Boba is built on the Optimistic Rollup developed by [Optimism](https://optimism.io). In addition to the native features of Optimism like bridging ETH and other tokens (including NFTs), Boba extends Optimism with compatible extensions including [Hybrid Compute](contents/hc/README.md) and [Boba as a Fee Token](contents/developer/features/aa-basics/aa-paymasters.md).  Boba is organized as a [DAO](https://gateway.boba.network/dao).

<figure><img src="./assets/documentation.png" alt=""><figcaption></figcaption></figure>

User focused documentation is available [here on the Boba docs website](http://docs.boba.network/). If you have questions or feel like something is missing check out our [Discord server](https://discord.com/invite/Hvu3zpFwWd) where we are actively responding, or [open an issue](https://github.com/bobanetwork/boba/issues) in the GitHub repo for this site.

### Direct Support

[Telegram for Developers](https://t.me/bobadev)\
[Project Telegram](https://t.me/bobanetwork)\
[Discord](https://discord.com/invite/Hvu3zpFwWd)

<figure><img src="./assets/directory-structure.png" alt=""><figcaption></figcaption></figure>

The basic directory structure is laid out [in the repository README.md](https://github.com/bobanetwork/boba?tab=readme-ov-file#directory-structure).

Additionally, you will find specific Boba directories for:

* [`boba-bindings`](https://github.com/bobanetwork/boba/tree/develop/boba-bindings): Go Bindings for the rollup smart contracts, including additional Boba contracts (like the DAO and the Boba Token).
* [`boba-chain-ops`](https://github.com/bobanetwork/boba/tree/develop/boba-chain-ops): Tooling created to migrate the state of the original Boba network to Boba Anchorage.
* [`boba-community`](https://github.com/bobanetwork/boba/tree/develop/boba-community): Easy to use docker-compose environments to execute a Boba replica.

<figure><img src="./assets/contributing.png" alt=""><figcaption></figcaption></figure>

Follow these instructions to set up your local development environment.

### Dependencies

You'll need the following:

* [Git](https://git-scm.com/downloads)
* [NodeJS](https://nodejs.org/en/download/)
* [Make](https://www.gnu.org/software/make/)
* [pnpm](https://pnpm.io/installation)
* [Docker](https://docs.docker.com/get-docker/)
* [Docker Compose](https://docs.docker.com/compose/install/)

**Note: this is only relevant to developers who wish to work on Boba core services. For most test uses, e.g. deploying your contracts, it's simpler to use the RPC provider at https://sepolia.boba.network**.

Clone the repository, open it, and build with `make`:

```bash
$ git clone https://github.com/bobanetwork/boba.git
$ cd boba
$ make
```

Then, make sure you have Docker installed _and make sure Docker is running_. Finally, build and run the entire stack:

```bash
$ make devnet-up
```

<figure><img src="./assets/spinning up the stack (1).png" alt=""><figcaption></figcaption></figure>

Stack spinup can take 15 minutes or more. There are many interdependent services to bring up with multiple sets of contract deployment and initialization.  More CPU cores and larger allocated RAM will help the process proceed more quickly.

Once the stack is up, you can verify its functionality with:

```
make devnet-test
```

<figure><img src="./assets/helpful commands.png" alt=""><figcaption></figcaption></figure>

* _Running out of space on your Docker, or having other having hard to debug issues_? Try running `docker system prune -a --volumes` and then rebuild the images.
* _To (re)build services_: `make devnet-clean && make devnet-up`

<figure><img src="./assets/running unit tests (1).png" alt=""><figcaption></figcaption></figure>

To run unit tests for a specific package:

```bash
cd packages/package-to-test
make test
```

<figure><img src="./assets/running integration tests (1).png" alt=""><figcaption></figcaption></figure>

Make sure you are in the `op-e2e` folder and then run

```bash
make
```

Even with parallelism enabled, these tests can take 10 or more minutes to
execute, depending on the speed of your machine.

<figure><img src="./assets/viewing docker container logs.png" alt=""><figcaption></figcaption></figure>

By default, the `docker-compose up` command will show logs from all services, and that can be hard to filter through. In order to view the logs from a specific service, you can run:

```bash
docker-compose logs --follow <service name>
```

<figure><img src="./assets/license.png" alt=""><figcaption></figcaption></figure>

Code in this repository is licensed under the [MIT
License](https://github.com/bobanetwork/boba/blob/develop/LICENSE) unless
stated otherwise.  See specific licensing details for projects outside this
repo including [op-geth](https://github.com/ethereum-optimism/op-geth) and
[op-erigon](https://github.com/bobanetwork/op-erigon) which both contain code
under the [GNU
GPLv3](https://gist.github.com/kn9ts/cbe95340d29fc1aaeaa5dd5c059d2e60) and
other licenses.
