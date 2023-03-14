---
title: Explorer and Indexer
lang: en-US
---

The next step is to be able to see what is actually happening in your blockchain.
One easy way to do this is to use [Blockscout](https://www.blockscout.com/).

## Prerequisites

### Archive mode

Blockscout expects to interact with an Ethereum execution client in [archive mode](https://www.alchemy.com/overviews/archive-nodes#archive-nodes).
To create such a node, follow the [directions to add a node](./getting-started.md#adding-nodes), but in the command you use to start `op-geth` replace:

```sh
	--gcmode=full \
```

with

```sh
	--gcmode=archive \
```

### Docker

The easiest way to run Blockscout is to use Docker.
Download and install [Docker engine](https://docs.docker.com/engine/install/#server).


## Installation and configuration

1. Clone the Blockscout repository.

   ```sh
   cd ~
   git clone https://github.com/blockscout/blockscout.git
   cd blockscout/docker-compose
   ```

1. Depending on the version of Docker you have, there may be an issue with the environment path.
   Run this command to fix it:

   ```sh
   ln -s `pwd`/envs ..
   ```

1. If `op-geth` in archive mode runs on a different computer or a port that isn't 8545, edit `docker-compose-no-build-geth.yml` to set `ETHEREUM_JSONRPC_HTTP_URL` to the correct URL.

1. Start Blockscout

   ```sh
   docker compose -f docker-compose-no-build-geth.yml up
   ```

## Usage

After the docker containers start, browse to http:// < *computer running Blockscout* > :4000 to view the user interface. 

You can also use the [API](https://docs.blockscout.com/for-users/api)
