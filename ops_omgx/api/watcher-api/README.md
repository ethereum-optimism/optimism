# Watcher API

> Mainnet Endpoint: https://api-watcher.mainnet.boba.network/
> Rinkeby Endpoint: https://api-watcher.rinkeby.boba.network/

## Methods

### get.transaction

**Request Body**

```js
{
  address: "ACCOUNT",
  from: "NUMBER",
  to: "NUMBER"
}
```

**Response Body**

```js
[
  {
    hash: "TRANSACTION_HASH",
    blockNumber: "BLOCK_NUMBER",
    from: "FROM_ACCOUNT",
    to: "TO_ACCOUNT",
    timestamp: "BLOCK_TIMESTAMP",
    exitL2: "EXIT_L2", // True or False
    crossDomainMessage: {
      crossDomainMessage: "CROSS_DOMAIN_MESSAGE", // whether the transaction sent cross domain message
      crossDomainMessageFinalize: "CROSS_DOMAIN_MESSAGE_FINALIZED", // whether the cross domain message is finalized on L1
      crossDomainMessageSendTime: "CROSS_DOMAIN_MESSAGE_FINALIZED_TIME", // when the cross domain message is finalized
      crossDomainMessageEstimateFinalizedTime: "ESTIMATE_CROSS_DOMAIN_MESSAGE_FINALIZED_TIME",
      fastRelay: "FAST_RELAY", // Whether the message is using the fast message relayer
      l1Hash: "L1_HASH", // L1 hash of the cross domain message
      l1BlockNumber: "L1_BLOCK_NUMBER",
      l1BlockHash: "L1_BLOCK_HASH",
      l1From: "L1_FROM",
      l1To: "L1_TO"
    },
    stateRoot: {
      stateRootHash: "L1_STATE_ROOT_HASH",
      stateRootBlockNumber: "L1_STATE_ROOT_BLOCK_NUMBER",
      stateRootBlockHash: "L1_STATE_ROOT_BLOCK_HASH",
      stateRootBlockTimestamp: "L1_STATE_ROOT_BLOCK_TIMESTAMP"
    },
    exit: {
      exitSender: "EXIT_SENDER", // The address of L2 token sender
      exitTo: "EXIT_RECEIVER", // The address of L1 token receiver
      exitToken: "EXIT_TOKEN", // L2 token address
      exitAmount: "EXIT_AMOUNT", // L2 exit amount, which doesn't consider fee
      exitReceive: "EXIT_RECEIVE", // L1 received amount
      exitFeeRate: "EXIT_FEE",
      fastRelay: "FAST_RELAY",
      status: "STATUS" // pending || succeeded || reverted
    }
  }
]
```

### get.deployments

**Request Body**

```js
{
  address: "ACCOUNT"
}
```

**Response Body**

```js
[
  {
    hash: "TRANSACTION_HASH",
    blockNumber: "BLOCK_NUMBER",
    from: "FROM_ACCOUNT",
    timeStamp: "BLOCK_TIMESTAMP",
    contractAddress: "CONTRACT_ADDRESS"
  }
]
```

### get.crossdomainmessage

**Request Body**

```js
{
  hash: "HASH"
}
```

**Response Body**

```js
{
  hash: "TRANSACTION_HASH",
  blockNumber: "BLOCK_NUMBER",
  from: "FROM_ACCOUNT",
  to: "TO_ACCOUNT"
  timeStamp: "BLOCK_TIMESTAMP",
  crossDomainMessage: "CROSS_DOMAIN_MESSAGE", // whether the transaction sent cross domain message
  crossDomainMessageFinalize: "CROSS_DOMAIN_MESSAGE_FINALIZED", // whether the cross domain message is finalized on L1
  crossDomainMessageSendTime: "CROSS_DOMAIN_MESSAGE_FINALIZED_TIME", // when the cross domain message is finalized
  crossDomainMessageEstimateFinalizedTime: "ESTIMATE_CROSS_DOMAIN_MESSAGE_FINALIZED_TIME",
  fastRelay: "FAST_RELAY" // Whether the message is using the fast message relayer
  l1Hash: "L1_HASH", // L1 hash of the cross domain message
  l1BlockNumber: "L1_BLOCK_NUMBER",
  l1BlockHash: "L1_BLOCK_HASH",
  l1From: "L1_FROM",
  l1To: "L1_TO"
}
```

