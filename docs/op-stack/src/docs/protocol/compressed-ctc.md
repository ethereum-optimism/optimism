---
title: The Cannonical Transaction Chain (CTC) Format
lang: en-US
---

Every transaction submitted to Optimism is written to the mainnet Ethereum blockchain as call data, this is how Optimism inherits the availability and integrity guarantees of Ethereum.
This is also the cause of the majority of the cost of Optimism transactions.
At the time of writing it is cheaper to write a kilobytes to storage on Optimism than it is to add one byte to the calldata on Ethereum.

## Initial solution

The initial solution was to write a header with supporting data followed by a list of transactions.

To interpret a transaction, you can search for it on [Etherscan](https://etherscan.io/).
To interpret a CTC transaction you need to **Click to see more** to see the calldata (called "Input Data" by Etherscan):

![Transaction input data](../../assets/docs/protocol/compressed-ctc/input-data.png)

For example, here are the fields and their values for this [initial solution CTC transaction](https://etherscan.io/tx/0xf5a2dd9d0815ad4dcee00063ff8f8f3fd44b3bd8ffc1f7f6c7f7f0b4b086c5a7/advanced):

| Bytes | Field Size | Field             | Value | Comments |
| ---------: | ---------: | ------------------| ----- | -------- |
|  0-3 |  4 | Function signature | 0xd0f89344 | [appendSequencerBatch()](https://www.4byte.directory/signatures/?bytes4_signature=0xd0f89344) |
|  4-8 |  5 | Starting tx index   | 4025992 | [this transaction](https://explorer.optimism.io/tx/4025992) |
|  9-11 |  3 | Elements to append | 89 |
| 12-14 |  3 | Batch contexts     | 15 |
| 15-30 | 14 | **Context 0** (multiple fields) |
| 15-17 |  3 | Transactions sent directly to L2 | 3 |
| 18-20 |  3 | Deposits with this context | 0 |
| 21-25 |  5 | Timestamp | 1646146436 | `block.timestamp` for transactions in this context (Tue Mar 01 2022 14:53:56 UTC)
| 26-30 |  5 | L1 block number | The L1 block number in this context, as obtained by calling [OVM_L1BlockNumber](https://github.com/ethereum-optimism/optimism/blob/develop/packages/contracts/contracts/L2/predeploys/iOVM_L1BlockNumber.sol). ([14301739](https://etherscan.io/block/14301739)) | 
| 31-46 | 14 | **Context 1** |
| 31-33 |  3 | Transactions sent directly to L2 | 8 |
| 34-36 |  3 | Deposits | 0 |
| 37-41 |  5 | Timestamp | 1646146451 | 15 seconds after the previous batch
| 42-47 | 5  | L1 block number | [14301739](https://etherscan.io/block/14301739) 
| 16n+15-16n+30 | 14 | **Context n** |
| 16n+15-16n+17 |  3 | Transactions sent directly to L2 |
| 16n+18-16n+20 |  3 | Deposits with this context
| 16n+21-16n+25 |  5 | Timestamp 
| 16n+26-16n+30 |  5 | L1 block number


This transaction has 15 batch contexts (numbered 0-14), so the first byte after the last context in this transaction, which starts the transaction list, is `16*15+15=255`.
Transactions are provided as a three byte length followed by the RLP encoded transaction.

Looking at locations 255-257, we see `0x00016e`, so the first transaction is 366 bytes long, at locations 258-623. 
The next transaction length starts at byte 624, and the next transaction itself starts at byte 627.

## Additional batch types

The initial solution did not have a batch type. 
However, to modify the format (for example, to add compression) a batch type is needed.
The solution is to set the timestamp of the first context to zero.
This does not create ambiguity because a timestamp of zero represents January 1st, 1970 (based on the UNIX convention), which cannot happen.
The block number then represents the transaction type.

## CTC transaction type zero

After the normal header and the first context, which has a timestamp of zero and a block number of zero, the other contexts contain the normal data. 
After that the list of transaction lengths and transaction data is compressed using [zlib](https://nodejs.org/api/zlib.html).
