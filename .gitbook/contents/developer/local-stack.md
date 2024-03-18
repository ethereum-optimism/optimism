---
description: Learn how to run the Boba development stack
---

# Development Stack

**Note: this is only relevant to developers who wish to work on Boba core services**.

For most test uses, it's simpler to use [https://goerli.boba.network](https://goerli.boba.network). Clone the repository, open it, and install nodejs packages with `yarn`:

```bash
$ git clone git@github.com:bobanetwork/boba.git
$ cd boba
$ yarn clean
$ yarn
$ yarn build
```

Then, make sure you have Docker installed _and make sure Docker is running_. Finally, build and run the entire stack:

```bash
$ cd ops
$ BUILD=1 DAEMON=0 ./up_local.sh
```



<figure><img src="../../.gitbook/assets/spinning up the stack.png" alt=""><figcaption></figcaption></figure>

Stack spinup can take 15 minutes or more. There are many interdependent services to bring up with two waves of contract deployment and initialisation. Recommended settings - 10 CPUs, 30 to 40 GB of memory. You can either inspect the Docker `Dashboard>Containers/All>Ops` for the progress of the `ops_deployer` _or_ you can run this script to wait for the sequencer to be fully up:

```
./scripts/wait-for-sequencer.sh
```

If the command returns with no log output, the sequencer is up. Once the sequencer is up, you can inspect the Docker `Dashboard>Containers/All>Ops` for the progress of `ops_boba_deployer` _or_ you can run the following script to wait for all the Boba contracts (e.g. the fast message relay system) to be deployed and up:

```
./scripts/wait-for-boba.sh
```

When the command returns with `Pass: Found L2 Liquidity Pool contract address`, the entire Boba stack has come up correctly.



<figure><img src="../../.gitbook/assets/hepful commands.png" alt=""><figcaption></figcaption></figure>

* _Running out of space on your Docker, or having other having hard to debug issues_? Try running `docker system prune -a --volumes` and then rebuild the images.
* _To (re)build individual base services_: `docker-compose build -- l2geth`
* _To (re)build individual Boba services_: `docker-compose -f "docker-compose.yml" build -- boba_message-relayer-fast` Note: First you will have to comment out various dependencies in `docker-compose.yml`.



<figure><img src="../../.gitbook/assets/running unit tests.png" alt=""><figcaption></figcaption></figure>

To run unit tests for a specific package:

```bash
cd packages/package-to-test
yarn test
```



<figure><img src="../../.gitbook/assets/running integration tests.png" alt=""><figcaption></figcaption></figure>

Make sure you are in the `ops` folder and then run

```bash
docker-compose run integration_tests
```

Expect the full test suite to complete in between _30 minutes_ to _two hours_ depending on your computer hardware.
