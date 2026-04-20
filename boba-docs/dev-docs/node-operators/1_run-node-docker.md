# Running a Node with Docker

This tutorial will walk you through the process of using Docker to run an BOBA Sepolia node, OP Mainnet node and OP Sepolia node. You can find all Docker Compose files [here](https://github.com/bobanetwork/boba/tree/develop/boba-community).

## Choose Your Execution Client

Boba supports multiple execution clients. Pick the one that suits your needs:

| Compose File | Execution Client | Consensus Client | Status |
|---|---|---|---|
| `docker-compose-boba-{network}-reth.yml` | **op-reth** | **op-node** | Recommended |
| `docker-compose-boba-{network}-geth.yml` | op-geth | op-node | Deprecated (removal 2026-05-31) |

> **op-reth** is the recommended execution client for new deployments. It uses the upstream [op-reth](https://github.com/paradigmxyz/reth) image with Boba chains built-in — no custom build is required.

## Prerequisites

* [docker](https://docs.docker.com/engine/install/)
* [docker-compose](https://docs.docker.com/compose/install/)

## Setup

Clone the `boba` repository to get started

```bash
git clone https://github.com/bobanetwork/boba.git
cd boba
cd boba-community
```

## Configuration

Configuration for the `docker-compose` is handled through environment variables inside of an `.env` file.

### Create an `.env` file

The repository includes a sample environment variable file located at `.env.example` that you can copy and modify to get started. Make a copy of this file and name it `.env`.

```bash
cp .env.example .env
```

### Configure the `.env` file

Open the `.env` in your directory and set the variables inside. Read the descriptions of each variable to understand what they do and how to set them. Read the [software release](software-release) page to set the correct version.

## DB Configuration

### Create a Shared Secret (JWT Token)

```bash
openssl rand -hex 32 > jwt-secret.txt
```

### Download Snapshots

Download the database snapshot for the client and network you wish to run. Always verify snapshots by comparing the sha256sum of the downloaded file to the sha256sum listed on the [snapshot downloads](snapshot-downloads) page.

```bash
sha256sum <filename>
```

#### op-reth (Recommended)

* BOBA Mainnet

  ```bash
  curl -o boba-mainnet-reth-db-20260320.tar.zst -sL https://boba-db.s3.us-east-2.amazonaws.com/mainnet/boba-mainnet-reth-db-20260320.tar.zst
  ```

* BOBA Sepolia

  ```bash
  curl -o boba-sepolia-reth-db-20260415.tar.zst -sL https://boba-db.s3.us-east-2.amazonaws.com/sepolia/boba-sepolia-reth-db-20260415.tar.zst
  ```

Extract the snapshot into a `reth-data` directory:

```bash
mkdir -p reth-data
tar --zstd -xf boba-{network}-reth-db-initial.tar.zst -C reth-data
```

These snapshots contain a pre-initialized reth database built from the op-geth state at the Bedrock migration block (block 1149019 for mainnet, block 511 for sepolia). No manual `init-state` step is needed — just extract and run. To regenerate the database from scratch, see the [migration guide](https://github.com/bobanetwork/boba/tree/develop/boba-community/scripts/geth-to-reth).

Set the `DATA_DIR` in your `.env` to point to this directory (or leave it blank to use the default `./reth-data`).

#### op-geth (Deprecated)

> **Deprecated:** op-geth support ends 2026-05-31. Migrate to op-reth.

See the [snapshot downloads](snapshot-downloads) page for available geth snapshots.

Extract geth snapshots:

```bash
tar xvf data.tgz
```

### Modify Volume Location

The volumes of l2 and op-node should be modified to your file locations.

```yaml
l2:
  volumes:
    - ./jwt-secret.txt:/config/jwt-secret.txt
    - DATA_DIR:/db
op-node:
  volumes:
  	- ./jwt-secret.txt:/config/jwt-secret.txt
```

## Run the Node

Once you've configured your `.env` file, you can run the node using Docker Compose.

### op-reth (Recommended)

```bash
# BOBA Mainnet
docker compose -f docker-compose-boba-mainnet-reth.yml up -d

# BOBA Sepolia
docker compose -f docker-compose-boba-sepolia-reth.yml up -d
```

### op-geth (Deprecated — removal 2026-05-31)

```bash
# BOBA Sepolia
docker compose -f docker-compose-boba-sepolia-geth.yml up -d

# BOBA Mainnet
docker compose -f docker-compose-boba-mainnet-geth.yml up -d
```

## Operating the Node

### Start

```bash
docker-compose -f [docker-compose-file] up -d
```

Will start the node in a detatched shell (`-d`), meaning the node will continue to run in the background.

### View Logs

```bash
docker-compose logs -f --tail 10
```

To view logs of all containers.

```bash
docker-compose logs <CONTAINER_NAME> -f --tail 10
```

### Stop

```bash
docker-compose -f [docker-compose-file] down
```

### Wipe [DANGER]

```bash
docker-compose -f [docker-compose-file] down -v
```

