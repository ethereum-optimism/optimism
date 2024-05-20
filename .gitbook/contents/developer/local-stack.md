---
description: Learn how to run the Boba development stack
---

# Development Stack

**Note: this is only relevant to developers who wish to work on Boba core services**.

For most test uses, it's simpler to use [https://sepolia.boba.network](https://sepolia.boba.network). In most cases, the contract deployment experience is exactly like deploying on Ethereum. You will need to have some testnet ETH and you will have to change your RPC endpoint to the Sepolia URL (or `https://mainnet.boba.network` for the production network). That's it!

The following instructions apply to those who do wish to run a local standalone system.

Prerequisites include:
* Version 1.21 of the Go language (https://go.dev/dl/)
* A current version of Node.js (https://nodejs.org)
* The Yarn package manager (https://yarnpkg.com/getting-started/install) - can be enabled after installing Node.js
* The pnpm package manager (https://pnpm.io)
* The jq JSON processor (https://jqlang.github.io/jq/ or 'sudo apt install jq')
* The Foundry toolkit (https://getfoundry.sh/)

Download and install them according to their respective instructions, including the steps to update your $PATH environment variable. Then clone and build the Boba repository.

```bash
$ git clone https://github.com/bobanetwork/boba.git
$ cd boba
$ make
```

```bash
To clean up and rebuild:
$ make nuke
$ make
```

<figure><img src="../../assets/spinning up the stack.png" alt=""><figcaption></figcaption></figure>

Make sure you have Docker installed _and make sure Docker is running_.

```bash
$ make devnet-hardhat-up
```

This will bring up the stack, including L1 and L2 sequencers as well as other components of the stack. Initial spinup can take 15 minutes or more as dependencies are downloaded, but subsequent relaunches will be faster.

Various setup files including a list of contract addresses may be found in the .devnet directory:
```bash
$ ls .devnet
addresses.json  allocs-l1.json  genesis-l1.json  genesis-l2.json  rollup.json  test-jwt-secret.txt
```

To stop the stack and delete its Docker containers:
```bash
$ make devnet-clean
```

<figure><img src="../../assets/hepful commands.png" alt=""><figcaption></figcaption></figure>

* _Running out of space on your Docker, or having other having hard to debug issues_? Try running `docker system prune -a --volumes` and then rebuild the images.

* The system may be inspected through Docker commands.
```bash
$ docker ps
CONTAINER ID   IMAGE                                                     COMMAND                  CREATED         STATUS         PORTS                                                                                                                                                                        NAMES
7f62bd783a1b   us-docker.pkg.dev/local/local/images/op-proposer:devnet   "op-proposer"            2 minutes ago   Up 2 minutes   0.0.0.0:6062->6060/tcp, :::6062->6060/tcp, 0.0.0.0:7302->7300/tcp, :::7302->7300/tcp, 0.0.0.0:6546->8545/tcp, :::6546->8545/tcp                                              ops-bedrock-op-proposer-1
960705b0fb65   us-docker.pkg.dev/local/local/images/op-batcher:devnet    "op-batcher"             2 minutes ago   Up 2 minutes   0.0.0.0:6061->6060/tcp, :::6061->6060/tcp, 0.0.0.0:7301->7300/tcp, :::7301->7300/tcp, 0.0.0.0:6545->8545/tcp, :::6545->8545/tcp                                              ops-bedrock-op-batcher-1
03e8e9ba5669   bobanetwork/local-kms:latest                              "local-kms"              2 minutes ago   Up 2 minutes   0.0.0.0:8888->8888/tcp, :::8888->8888/tcp                                                                                                                                    ops-bedrock-kms-1
7c3321df29a3   us-docker.pkg.dev/local/local/images/op-node:devnet       "op-node --l1=ws://l\u2026"   2 minutes ago   Up 2 minutes   0.0.0.0:6060->6060/tcp, :::6060->6060/tcp, 0.0.0.0:7300->7300/tcp, :::7300->7300/tcp, 0.0.0.0:9003->9003/tcp, :::9003->9003/tcp, 0.0.0.0:7545->8545/tcp, :::7545->8545/tcp   ops-bedrock-op-node-1
d2dcc611348e   ops-bedrock-l2                                            "/bin/sh /home/boba/\u2026"   2 minutes ago   Up 2 minutes   8080/tcp, 8546/tcp, 8551/tcp, 9090/tcp, 30303/tcp, 30303/udp, 42069/tcp, 42069/udp, 0.0.0.0:8060->6060/tcp, :::8060->6060/tcp, 0.0.0.0:9545->8545/tcp, :::9545->8545/tcp     ops-bedrock-l2-1
0407faa2e335   ops-bedrock-l1                                            "/bin/bash /entrypoi\u2026"   7 minutes ago   Up 7 minutes   0.0.0.0:8545-8546->8545-8546/tcp, :::8545-8546->8545-8546/tcp, 30303/tcp, 30303/udp, 0.0.0.0:7060->6060/tcp, :::7060->6060/tcp                                               ops-bedrock-l1-1

$ cd ops-bedrock
$ docker compose logs --follow op-node # or any other system component
```

<figure><img src="../../assets/running unit tests.png" alt=""><figcaption></figcaption></figure>

To run unit tests for a specific Optimism service:

```bash
cd op-node # or op-proposer, etc.
make test
```

<figure><img src="../../assets/running integration tests.png" alt=""><figcaption></figcaption></figure>

To run end-to-end tests:

```bash
$ cd op-e2e
$ make test
```

<figure><img src="../../assets/deploying standard contracts.png" alt=""><figcaption></figcaption></figure>

The L2 RPC endpoint is `https://127.0.0.1:9545`. The local L1 may be accessed through `https://127.0.0.1:8545`.
