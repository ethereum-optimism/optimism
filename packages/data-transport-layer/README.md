# @eth-optimism/data-transport-layer

[![codecov](https://codecov.io/gh/ethereum-optimism/optimism/branch/develop/graph/badge.svg?token=0VTG7PG7YR&flag=dtl-tests)](https://codecov.io/gh/ethereum-optimism/optimism)

## What is this?

The Optimism Data Transport Layer is a long-running software service (written in TypeScript) designed to reliably index Optimism transaction data from Layer 1 (Ethereum). Specifically, this service indexes:

* Transactions that have been enqueued for submission to the CanonicalTransactionChain via [`CanonicalTransactionChain.enqueue`].
* Transactions that have been included in the CanonicalTransactionChain via [`CanonicalTransactionChain.appendQueueBatch`] or [`CanonicalTransactionChain.appendSequencerBatch`].
* State roots (transaction results) that have been published to the StateCommitmentChain via [`StateCommitmentChain.appendStateBatch`].

## How does it work?

We run two sub-services, the [`L1IngestionService`](./src/services/l1-ingestion/service.ts) and the [`L1TransportServer`](./src/services/server/service.ts). The `L1IngestionService` is responsible for querying for the various events and transaction data necessary to accurately index information from our Layer 1 (Ethereum) smart contracts. The `L1TransportServer` simply provides an API for accessing this information.

## Getting started

### Configuration

See an example config at [.env.example](.env.example); copy into a `.env` file before running.

`L1_TRANSPORT__L1_RPC_ENDPOINT` can be the JSON RPC endpoint of any L1 Ethereum node. `L1_TRANSPORT__ADDRESS_MANAGER` should be the contract addresss of the Address Manager on the corresponding network; find their values in the [contracts package](https://github.com/ethereum-optimism/optimism/tree/develop/packages/contracts/deployments).

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

## Configuration

We're using `dotenv` for our configuration.
Copy `.env.example` into `.env`, feel free to modify it.
Here's the list of environment variables you can change:

| Variable                                                | Default     | Description                                                                                                                                                   |
| ------------------------------------------------------- | ----------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| DATA_TRANSPORT_LAYER__DB_PATH                           | ./db        | Path to the database for this service.                                                                                                                        |
| DATA_TRANSPORT_LAYER__ADDRESS_MANAGER                   | -           | Address of the AddressManager contract on L1. See [contracts](https://github.com/ethereum-optimism/optimism/tree/develop/packages/contracts/deployments) package to find this address for mainnet or kovan. |
| DATA_TRANSPORT_LAYER__POLLING_INTERVAL                  | 5000        | Period of time between execution loops.                                                                                                                       |
| DATA_TRANSPORT_LAYER__DANGEROUSLY_CATCH_ALL_ERRORS      | false       | If true, will catch all errors without throwing.                                                                                                              |
| DATA_TRANSPORT_LAYER__CONFIRMATIONS                     | 12          | Number of confirmations to wait before accepting transactions as "canonical".                                                                                 |
| DATA_TRANSPORT_LAYER__SERVER_HOSTNAME                   | localhost   | Host to run the API on.                                                                                                                                       |
| DATA_TRANSPORT_LAYER__SERVER_PORT                       | 7878        | Port to run the API on.                                                                                                                                       |
| DATA_TRANSPORT_LAYER__SYNC_FROM_L1                      | true        | Whether or not to sync from L1.                                                                                                                               |
| DATA_TRANSPORT_LAYER__L1_RPC_ENDPOINT                   | -           | RPC endpoint for an L1 node.                                                                                                                                  |
| DATA_TRANSPORT_LAYER__L1_RPC_USER                       | -           | Basic Authentication user for the L1 node endpoint.                                                                                                           |
| DATA_TRANSPORT_LAYER__L1_RPC_PASSWORD                   | -           | Basic Authentication password for the L1 node endpoint.                                                                                                       |
| DATA_TRANSPORT_LAYER__LOGS_PER_POLLING_INTERVAL         | 2000        | Logs to sync per polling interval.                                                                                                                            |
| DATA_TRANSPORT_LAYER__SYNC_FROM_L2                      | false       | Whether or not to sync from L2.                                                                                                                               |
| DATA_TRANSPORT_LAYER__L2_RPC_ENDPOINT                   | -           | RPC endpoint for an L2 node.                                                                                                                                  |
| DATA_TRANSPORT_LAYER__L2_RPC_USER                       | -           | Basic Authentication user for the L2 node endpoint.                                                                                                           |
| DATA_TRANSPORT_LAYER__L2_RPC_PASSWORD                   | -           | Basic Authentication password for the L2 node endpoint.                                                                                                       |
| DATA_TRANSPORT_LAYER__TRANSACTIONS_PER_POLLING_INTERVAL | 1000        | Number of L2 transactions to query per polling interval.                                                                                                      |
| DATA_TRANSPORT_LAYER__L2_CHAIN_ID                       | -           | L2 chain ID.                                                                                                                                                  |
| DATA_TRANSPORT_LAYER__LEGACY_SEQUENCER_COMPATIBILITY    | false       | Whether or not to enable "legacy" sequencer sync (without the custom `eth_getBlockRange` endpoint)                                                            |
| DATA_TRANSPORT_LAYER__NODE_ENV                          | development | Environment the service is running in: production, development, or test.                                                                                      |
| DATA_TRANSPORT_LAYER__ETH_NETWORK_NAME                  | -           | L1 Ethereum network the service is deployed to: mainnet, kovan, goerli.                                                                                  |
| DATA_TRANSPORT_LAYER__L1_GAS_PRICE_BACKEND                  | l1           | Where to pull the l1 gas price from (l1 or l2)                                                                                  |
| DATA_TRANSPORT_LAYER__DEFAULT_BACKEND                  | l1           | Where to sync transactions from (l1 or l2)                                                                                  |

To enable proper error tracking via Sentry on deployed instances, make sure `NODE_ENV` and `ETH_NETWORK_NAME` are set in addition to [`SENTRY_DSN`](https://docs.sentry.io/platforms/node/).

## HTTP API

This section describes the HTTP API for accessing indexed Layer 1 data.

### Latest Ethereum Block Context

#### Request

```
GET /eth/context/latest
```

#### Response

```ts
{
    "blockNumber": number,
    "timestamp": number
}
```

### Latest Ethereum L1 Gas Price

#### Request

```
GET /eth/gasprice
```
Defaults to pulling L1 gas price from config option `DATA_TRANSPORT_LAYER__L1_GAS_PRICE_BACKEND`. Can be overridden with query parameter `backend` (`/eth/gasprice?backend=l1`).

#### Response

```ts
{
    "gasPrice": string
}
```


### Enqueue by Index

#### Request

```
GET /enqueue/index/{index: number}
```

#### Response

```ts
{
  "index": number,
  "target": string,
  "data": string,
  "gasLimit": number,
  "origin": string,
  "blockNumber": number,
  "timestamp": number
}
```

### Transaction by Index

#### Request

```
GET /transaction/index/{index: number}
```

#### Response

```ts
{
    "transaction": {
        "index": number,
        "batchIndex": number,
        "data": string,
        "blockNumber": number,
        "timestamp": number,
        "gasLimit": number,
        "target": string,
        "origin": string,
        "queueOrigin": string,
        "type": string | null,
        "decoded": {
            "sig": {
                "r": string,
                "s": string,
                "v": string
            },
            "gasLimit": number,
            "gasPrice": number,
            "nonce": number,
            "target": string,
            "data": string
        } | null,
        "queueIndex": number | null,
    },

    "batch": {
        "index": number,
        "blockNumber": number,
        "timestamp": number,
        "submitter": string,
        "size": number,
        "root": string,
        "prevTotalElements": number,
        "extraData": string
    }
}
```

### Transaction Batch by Index

#### Request

```
GET /batch/transaction/index/{index: number}
```

#### Response

```ts
{
    "batch": {
        "index": number,
        "blockNumber": number,
        "timestamp": number,
        "submitter": string,
        "size": number,
        "root": string,
        "prevTotalElements": number,
        "extraData": string
    },

    "transactions": [
      {
        "index": number,
        "batchIndex": number,
        "data": string,
        "blockNumber": number,
        "timestamp": number,
        "gasLimit": number,
        "target": string,
        "origin": string,
        "queueOrigin": string,
        "type": string | null,
        "decoded": {
            "sig": {
                "r": string,
                "s": string,
                "v": string
            },
            "gasLimit": number,
            "gasPrice": number,
            "nonce": number,
            "target": string,
            "data": string
        } | null,
        "queueIndex": number | null,
      }
    ]
}
```


### State Root by Index

#### Request

```
GET /stateroot/index/{index: number}
```

#### Response

```ts
{
    "stateRoot": {
        "index": number,
        "batchIndex": number,
        "value": string
    },

    "batch": {
        "index": number,
        "blockNumber": number,
        "timestamp": number,
        "submitter": string,
        "size": number,
        "root": string,
        "prevTotalElements": number,
        "extraData": string
    },
}
```

### State Root Batch by Index

#### Request

```
GET /batch/stateroot/index/{index: number}
```

#### Response

```ts
{
    "batch": {
        "index": number,
        "blockNumber": number,
        "timestamp": number,
        "submitter": string,
        "size": number,
        "root": string,
        "prevTotalElements": number,
        "extraData": string
    },

    "stateRoots": [
        {
            "index": number,
            "batchIndex": number,
            "value": string
        }
    ]
}
```
