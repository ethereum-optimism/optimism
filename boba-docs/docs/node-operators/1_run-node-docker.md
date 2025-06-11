# Running a Node with Docker

This tutorial will walk you through the process of using Docker to run an BOBA Sepolia node, OP Mainnet node and OP Sepolia node. You can find all Docker Compose files [here](https://github.com/bobanetwork/boba/tree/develop/boba-community).

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

### Download Snapshots

You can download the database snapshot for the client and network you wish to run.

Always verify snapshots by comparing the sha256sum of the downloaded file to the sha256sum listed on this [page](snapshot-downloads). Check the sha256sum of the downloaded file by running `sha256sum <filename>`in a terminal.

* BOBA Mainnet

  The **erigon** db can be downloaded from the [boba mainnet erigon db](https://boba-db.s3.us-east-2.amazonaws.com/mainnet/boba-mainnet-erigon-db-1149019.tgz).

  ```bash
  curl -o boba-mainnet-erigon-db-1149019.tgz -sL https://boba-db.s3.us-east-2.amazonaws.com/mainnet/boba-mainnet-erigon-db-1149019.tgz
  ```

  The **geth** db can be downloaded from [boba mainnet geth db](https://boba-db.s3.us-east-2.amazonaws.com/mainnet/boba-mainnet-geth-db-114909.tgz).

  ```bash
  curl -o boba-mainnet-geth-db-114909.tgz -sL https://boba-db.s3.us-east-2.amazonaws.com/mainnet/boba-mainnet-geth-db-114909.tgz
  ```

- BOBA Sepolia

  The **erigon** db can be downloaded from the [boba sepolia erigon db](https://boba-db.s3.us-east-2.amazonaws.com/sepolia/boba-sepolia-erigon-db.tgz).

  ```bash
  curl -o boba-sepolia-erigon-db.tgz -sL https://boba-db.s3.us-east-2.amazonaws.com/sepolia/boba-sepolia-erigon-db.tgz
  ```

  The **geth** db can be downloaded from [boba sepolia geth db](https://boba-db.s3.us-east-2.amazonaws.com/sepolia/boba-sepolia-geth-db.tgz).

  ```bash
  curl -o boba-sepolia-geth-db.tgz -sL https://boba-db.s3.us-east-2.amazonaws.com/sepolia/boba-sepolia-geth-db.tgz
  ```

- OP Mainnet

  The **erigon** db can be downloaded from [Test in Prod OP Mainnet](https://op-erigon-backup.mainnet.testinprod.io).

- OP Sepolia

  The **erigon** db can be downloaded from [optimism sepolia erigon db](https://boba-db.s3.us-east-2.amazonaws.com/sepolia/optimism-sepolia-erigon-db.tgz).

  Or you can download the genesis file from [Optimsim](https://networks.optimism.io/op-sepolia/genesis.json) and initialize the data directory with it.

  ```bash
  curl -o op-sepolia-genesis.json -sL https://networks.optimism.io/op-sepolia/genesis.json
  erigon init --datadir=/db genesis.json
  ```

  The erigon can be built from the [source](https://github.com/bobanetwork/v3-erigon) using `make erigon` .


### Extract Snapshots

Once you've downloaded the database snapshot, you'll need to extract it to a directory on your machine. This will take some time to complete.

```bash
tar xvf data.tgz
```

### Create a Shared Secret (JWT Token)

```bash
openssl rand -hex 32 > jwt-secret.txt
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

Once you've configured your `.env` file, you can run the node using Docker Compose. The following command will start the node in the background.

```bash
docker-compose -f [docker-compose-file] up -d
```

## Optional: Run the Node with Geth

We support both geth and erigon as the execution engines for Boba Mainnet node. You can start the node with geth using the following command:

```bash
docker-compose -f docker-compose-mainnet-geth.yml up -d
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

