# @eth-optimism/replica-healthcheck

## What is this?

`replica-healthcheck` is an express server to be run alongside a replica instance, to ensure that the replica is healthy. Currently, it exposes metrics on syncing stats and exits when the replica has a mismatched state root against the sequencer.

## Getting started

### Building and usage

After cloning and switching to the repository, install dependencies:

```bash
$ yarn
```

Use the following commands to build, use, test, and lint:

```bash
$ yarn build
$ yarn start
$ yarn test
$ yarn lint
```

### Configuration

We're using `dotenv` for our configuration.
To configure the project, clone this repository and copy the `env.example` file to `.env`.
Here's a list of environment variables:

| Variable                                        | Purpose                                                                                          | Default                                                                                  |
| ----------------------------------------------- | ------------------------------------------------------------------------------------------------ | ---------------------------------------------------------------------------------------- |
| REPLICA_HEALTHCHECK\_\_ETH_NETWORK              | Ethereum Layer1 and Layer2 network (mainnet,kovan)                                               | mainnet (change to `kovan` for the test network)                                         |
| REPLICA_HEALTHCHECK\_\_ETH_NETWORK_RPC_PROVIDER | Layer2 source of truth endpoint, used for the sync check                                         | https://mainnet.optimism.io (change to `https://kovan.optimism.io` for the test network) |
| REPLICA_HEALTHCHECK\_\_ETH_REPLICA_RPC_PROVIDER | Layer2 local replica endpoint, used for the sync check                                           | http://localhost:9991                                                                    |
| REPLICA_HEALTHCHECK\_\_L2GETH_IMAGE_TAG         | L2geth version                                                                                   | 0.4.9                                                                                    |
| REPLICA_HEALTHCHECK\_\_CHECK_TX_WRITE_LATENCY   | Boolean for whether to perform the transaction latency check. Recommend to only use for testnets | false                                                                                    |
| REPLICA_HEALTHCHECK\_\_WALLET1_PRIVATE_KEY      | Private key to one wallet for checking write latency                                             | -                                                                                        |
| REPLICA_HEALTHCHECK\_\_WALLET2_PRIVATE_KEY      | Private key to the other wallet for checking write latency                                       | -                                                                                        |
