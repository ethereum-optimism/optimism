# Running a Network Node

This project lets you set up a local replica of the Optimistic Ethereum chain (either the main one or the Kovan testnet). 
To submit transactions via a replica, set `SEQUENCER_CLIENT_HTTP` to a sequencer URL.

## Architecture

You need two components to replicate Optimistic Ethereum:

- `data-transport-layer`, which retrieves and indexes blocks from L1. 
  To access L1 you need an Ethereum Layer 1 provider, [such as one of those that provide both L1 and Optimism](https://community.optimism.io/docs/useful-tools/providers/).

- `l2geth`, which provides an Ethereum node where you applications can connect and run API calls.

## Resource requirements

The `data-transport-layer` should run with 1 CPU and 256Mb of memory.

The `l2geth` process should run with 1 or 2 CPUs and between 4 and 8Gb of memory.

With this configuration a synchronization from block 0 to current height is expect to take about 8 hours.

## Software Packages

These packages are required to run the replica:

1. [Docker](https://docs.docker.com/engine/install/)
1. [Docker compose](https://docs.docker.com/compose/install/)

## Configuration

To configure the project:

1. Clone this repository 

   ```sh
   git clone https://github.com/ethereum-optimism/optimism.git
   cd optimism/infra/op-replica/docker-compose/
   ```

1. Copy either `default-kovan.env` or `default-mainnet.env` file to `.env`.

   ```sh
   cp default-mainnet.env .env
   ```

### Settings

Edit the settings in `.env`.

```sh
nano .env
```
  
   
Change any other settings required for your environment

| Variable                 | Purpose                                                  | Default
| ------------------------ | -------------------------------------------------------- | -----------
| COMPOSE_FILE             | The yml files to use with docker-compose                 | replica.yml:replica-shared.yml
| ETH_NETWORK              | Ethereum Layer1 and Layer2 network (mainnet,kovan)       | Depends on the configuration file you use
| DATA_TRANSPORT_LAYER__L1_RPC_ENDPOINT | An endpoint for the L1 network, either kovan or mainnet. You **must** change this to a valid value.
| DATA_TRANSPORT_LAYER__L2_RPC_ENDPOINT | [Optimistic endpoint](https://community.optimism.io/docs/useful-tools/networks/), such as https://kovan.optimism.io or https://mainnet.optimism.io
| REPLICA_HEALTHCHECK__ETH_NETWORK_RPC_PROVIDER | The L2 endpoint to check the replica against | (typically the same as the DATA_TRANSPORT_LAYER__L2_RPC_ENDPOINT)
| SEQUENCER_CLIENT_HTTP | The L2 sequencer to forward tx to  | (typically the same as the DATA_TRANSPORT_LAYER__L2_RPC_ENDPOINT)
| SHARED_ENV_PATH          | Path to a directory containing env files                 | [a directory under .../op-replica](https://github.com/ethereum-optimism/optimism/tree/develop/infra/op-replica/envs)
| GCMODE                   | [Whether to run l2geth as an `archive` or `full` node](https://www.quicknode.com/guides/infrastructure/ethereum-full-node-vs-archive-node)     | archive
| L2GETH_IMAGE_TAG         | L2geth version                                           | [Go here](https://hub.docker.com/r/ethereumoptimism/l2geth/tags) and find the latest version. At writing this is 0.5.19 (1).
| DTL_IMAGE_TAG            | Data transport layer version                             | [Go here](https://hub.docker.com/r/ethereumoptimism/data-transport-layer/tags) and find the latest version. At writing this is 0.5.30 (1).
| HC_IMAGE_TAG             | Health check version                                     | [Go here](https://hub.docker.com/r/ethereumoptimism/data-transport-layer/tags) and find the latest version. At writing this is 1.0.6 (1).
| L2GETH_HTTP_PORT         | Port number for the l2geth RPC endpoint                  | 9991
| L2GETH_WS_PORT           | Port number for the l2geth WebSockets endpoint           | 9992
| DTL_PORT                 | Port number for the DTL endpoint, for troubleshooting    | 7878
| GETH_INIT_SCRIPT         | The script name to run when initializing l2geth          | A file under [op-replica/scripts](https://github.com/ethereum-optimism/optimism/tree/develop/infra/op-replica/envs)

Notes:

(1) It is easier to debug problems if you have the explicit version number, such as 0.5.19, instead of `latest`.

## Usage


| Action | Command |
| - | - |
| Start the replica (after which you can access it at `http://localhost:L2GETH_HTTP_PORT` | `docker compose up -d` |
| Get the logs for `l2geth` | `docker compose logs -f l2geth-replica` |
| Get the logs for `data-transport-layer` | `docker compose logs -f data-transport-layer` |
| Stop the replica | `docker compose down` |

The files the docker containers use are under `/var/replica`.


## Sync Check
 
There is a sync check container. It fails at startup because at that point the replica is not running yet. It exposes metrics on port 3000, which you could pick up with a Prometheus. You can view its status with this command:

```sh
docker compose logs -f replica-healthcheck
```

## Registration

[Register here](https://groups.google.com/a/optimism.io/g/optimism-announce) to get announcements, such as notifications of when you're supposed to update your replica.
