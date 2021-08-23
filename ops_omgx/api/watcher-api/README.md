# Watcher API

> Mainnet Endpoint: https://api-watcher.mainnet.omgx.network/
> Rinkeby Endpoint: https://api-watcher.rinkeby.omgx.network/

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
    timeStamp: "BLOCK_TIMESTAMP",
    crossDomainMessage: "CROSS_DOMAIN_MESSAGE", // whether the transaction sent cross domain message
    crossDomainMessageFinailze: "CROSS_DOMAIN_MESSAGE_FINALIZED", // whether the cross domain message is finalized on L1
    crossDomainMessageSendTime: "CROSS_DOMAIN_MESSAGE_FINALIZED_TIME", // when the cross domain message is finalized
    crossDomainMessageEstimateFinalizedTime: "ESTIMATE_CROSS_DOMAIN_MESSAGE_FINALIZED_TIME",
    fastRelay: "FAST_RELAY", // Whether the message is using the fast message relayer
    l1Hash: "L1_HASH", // L1 hash of the cross domain message
    l1BlockNumber: "L1_BLOCK_NUMBER",
    l1BlockHash: "L1_BLOCK_HASH",
    l1From: "L1_FROM",
    l1To: "L1_TO"
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
  crossDomainMessageFinailze: "CROSS_DOMAIN_MESSAGE_FINALIZED", // whether the cross domain message is finalized on L1
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

