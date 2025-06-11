# Bundler API

## Bundler Endpoints

|            |                                                                                      |
| ---------- | ------------------------------------------------------------------------------------ |
| ChainID    | 9728                                                                                 |
| AA bundler | [https://bundler.testnet.bnb.boba.network](https://bundler.testnet.bnb.boba.network) |

## Bundler API

This section lists the Ethereum JSON-RPC API endpoints for a basic EIP-4337 "bundler".

* `eth_sendUserOperation`
* `eth_supportedEntryPoints`
* `eth_chainId`
* `eth_estimateUserOperationGas`

### `eth_sendUserOperation`

Submit your userOperations to the bundler.

#### Parameters

1. `UserOperation`, a full user operation struct.
2. `EntryPoint`, address the request should be sent through.

#### Return value

Returns `userOpHash` if the UserOperation is valid.

Otherwise it returns an error object with `code` and `message`. (and sometimes `data`)

| **Code** | **Message**                                                                                                                         |
| -------- | ----------------------------------------------------------------------------------------------------------------------------------- |
| -32602   | Invalid UserOperation struct/fields                                                                                                 |
| -32500   | Transaction rejected by entryPoint's simulateValidation, during wallet creation or validation                                       |
| -32501   | Transaction rejected by paymaster's validatePaymasterUserOp                                                                         |
| -32502   | Transaction rejected because of opcode validation                                                                                   |
| -32503   | UserOperation out of time-range: either wallet or paymaster returned a time-range, and it is already expired (or will expire soon)  |
| -32504   | Transaction rejected because paymaster (or signature aggregator) is throttled/banned                                                |
| -32505   | Transaction rejected because paymaster (or signature aggregator) stake or unstake-delay is too low                                  |
| -32506   | Transaction rejected because wallet specified unsupported signature aggregator                                                      |
| -32507   | Transaction rejected because of wallet signature check failed (or paymaster siganture, if the paymaster uses its data as signature) |
| -32508   | UserOperation not in valid time-range: either wallet or paymaster returned a time-range, and it is valid in the future              |

#### Usage

Example request

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "eth_sendUserOperation",
  "params": [
    {
      sender, // address
      nonce, // uint256
      initCode, // bytes
      callData, // bytes
      callGasLimit, // uint256
      verificationGasLimit, // uint256
      preVerificationGas, // uint256
      maxFeePerGas, // uint256
      maxPriorityFeePerGas, // uint256
      paymasterAndData, // bytes
      signature // bytes
    },
    entryPoint // address
  ]
}
```

Example response:

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": "0x1234...5678"
}
```

Example failure response:

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "error": {
    "message": "paymaster stake too low",
    "data": {
      "paymaster": "0x123456789012345678901234567890123456790",
      "minimumStake": "0xde0b6b3a7640000",
      "minimumUnstakeDelay": "0x15180"
    },
    "code": -32504
  }
}
```

### `eth_supportedEntryPoints`

Returns an array of the entryPoint addresses supported by the client.

Request:

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "eth_supportedEntryPoints",
  "params": []
}
```

Response:

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": [
    "0xcd01C8aa8995A59eB7B2627E69b40e0524B5ecf8",
    "0x7A0A0d159218E6a2f407B99173A2b12A6DDfC2a6"
  ]
}
```

### `eth_chainId`

Returns EIP-155 Chain ID.

Request:

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "eth_chainId",
  "params": []
}
```

Response:

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": "0x1"
}
```

### `eth_getUserOperationByHash`

Get a UserOperation based on a userOperation hash

#### Parameters

1. `userOpHash`, a userOperation hash value

#### Return Value

Returns a full UserOperation, with the addition of `entryPoint`, `blockNumber`, `blockHash` and `transactionHash` if the UserOperation is included in a block.

Otherwise it returns `null` if the operation is yet to be included. For an invalid userOpHash returns an error object with `code`: -32601 and `message`: Missing/invalid userOpHash

Example request

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "eth_getUserOperationByHash",
  "params": [
    "0x1234...5678" // userOpHash
  ]
}
```

Example response:

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": [
    [
      "0xcd01C8aa8995A59eB7B2627E69b40e0524B5ecf8", // sender
      "1", // nonce
      "0x032...4324", // initCode
      "0x032...2131", // callData
      "500000", // callGasLimit
      "500000", // verificationGasLimit
      "500000", // preVerificationGas
      "500000", // maxFeePerGas
      "500000", // maxPriorityFeePerGas
      "0xa23...1231", // paymasterAndData
      "2312...31233" // signature
    ],
    "0xcd01C8aa8995A59eB7B2627E69b40e0524B5ecf8", //entryPoint
    "0x565738716e198cead3ce67d0ee33c28a9cfde6c2a70a602a87078c375cf98c7f", //txHash
    "0x9d15b109b62ec8ab8bbbf36650ceaca9666216c7f3a9d4bfece8cb06bdcd2422", //blockHash
    "12312" //blocknum
  ]
}
```

Example failure response:

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "error": {
    "message": "Missing/invalid userOpHash",
    "code": -32601
  }
}
```

### `eth_getUserOperationReceipt`

Get a UserOperation based on a userOperation hash

#### Parameters

1. `userOpHash`, a userOperation hash value

#### Return value

Returns a receipt that includes

`userOpHash`, the request hash `sender` `nonce` `actualGasCost`, actual amount paid (by account or paymaster) for this UserOperation `actualGasUsed`, total gas used by this UserOperation (including preVerification, creation, validation and execution) `success`, boolean - if this execution completed without revert `logs`, the logs generated by this UserOperation (not including logs of other UserOperations in the same bundle) `receipt`, the TransactionReceipt object.

Otherwise it returns `null` if the operation is yet to be included.

Example request

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "eth_getUserOperationReceipt",
  "params": [
    "0x1234...5678" // userOpHash
  ]
}
```

Example response:

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": [
    "0x1234...5678", //userOpHash
    "0xcd01C8aa8995A59eB7B2627E69b40e0524B5ecf8", //sender
    "2", //nonce
    "5000", //actual gas cost
    "5000", //actual gas used
    "true", //success
    [...], //logs
    [...], //receipt
  ]
}
```
