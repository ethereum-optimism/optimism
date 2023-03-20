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

### GraphQL

Blockscout's API includes [GraphQL](https://graphql.org/) support under `/graphiql`. 
For example, this query looks at addresses.

```
query {
  addresses(hashes:[
   "0xcB69A90Aa5311e0e9141a66212489bAfb48b9340", 
   "0xC2dfA7205088179A8644b9fDCecD6d9bED854Cfe"])
```

GraphQL queries start with a top level entity (or entities).
In this case, our [top level query](https://docs.blockscout.com/for-users/api/graphql#queries) is for multiple addresses.

Note that you can only query on fields that are indexed.
For example, here we query on the addresses.
However, we couldn't query on `contractCode` or `fetchedCoinBalance`.

```
 {
    hash
    contractCode
    fetchedCoinBalance
```

The fields above are fetched from the address table.

```
    transactions(first:5) {
```

We can also fetch the transactions that include the address (either as source or destination).
The API does not let us fetch an unlimited number of transactions, so here we ask for the first 5.


```
      edges {
        node {
```

Because this is a [graph](https://en.wikipedia.org/wiki/Graph_(discrete_mathematics)), the entities that connect two types, for example addresses and transactions, are called `edges`.
At the other end of each edge there is a transaction, which is a separate `node`.

```
          hash
          fromAddressHash
          toAddressHash
          input
        }
```

These are the fields we read for each transaction. 

```
      }
    }
  }
}
```

Finally, close all the brackets. 