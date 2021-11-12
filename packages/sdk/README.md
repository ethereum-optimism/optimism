[![codecov](https://codecov.io/gh/ethereum-optimism/optimism/branch/master/graph/badge.svg?token=0VTG7PG7YR&flag=sdk)](https://codecov.io/gh/ethereum-optimism/optimism)

# @eth-optimism/sdk

The `@eth-optimism/sdk` package provides a set of tools for interacting with Optimistic Ethereum.

## Installation

```
npm install @eth-optimism/sdk
```

## API

### Watcher

The `Watcher` class is a utility for observing transactions being sent between Ethereum (L1) and Optimistic Ethereum (L2).
You can use the `Watcher` to find when a message has been sent from one network and to find the transactions that execute those messages on the other network.

### Options

```typescript
interface WatcherOptions {
  l1: {
    // Ethers provider connected to L1
    provider: Provider
    // Address of the L1CrossDomainMessenger contract
    messengerAddress: string
  }
  l2: {
    // Ethers provider connected to L2
    provider: Provider
    // Address of the L2CrossDomainMessenger contract
    messengerAddress: string
  }
  // When using Watcher methods, whether or not to poll for pending transactions by default
  pollForPending?: boolean
  // When polling, number of milliseconds to wait between polls
  pollInterval?: number
}
```

### Example Usage

```typescript
import { Watcher } from '@eth-optimism/sdk'

// Set up an Ethers provider connected to L1
const l1Provider = ...

// Set up an Ethers provider connected to L2
const l2Provider = ...

// Set the L1CrossDomainMessenger address
// Use the deployments folder if connecting to a prod network:
// https://github.com/ethereum-optimism/optimism/tree/develop/packages/contracts/deployments
const l1MessengerAddress = ...

// Set the L2CrossDomainMessenger address
// This address is always 0x4200000000000000000000000000000000000007 on Optimistic Ethereum
// However, this address may be different on forks of OE
const l2MessengerAddress = 0x4200000000000000000000000000000000000007

const watcher = new Watcher({
  l1: {
    provider: l1Provider,
    messengerAddress: l1MessengerAddress,
  },
  l2: {
    provider: l2Provider,
    messengerAddress: l2MessengerAddress,
  },
})

// Have fun!
```

### Functions

#### getMessageHashesFromL1Tx

```typescript
/**
  * Pulls all L1 => L2 message hashes out of an L1 transaction by hash.
  *
  * @param l1TxHash Hash of the L1 transaction to find messages for.
  * @returns List of message hashes emitted in the transaction.
  */
public async getMessageHashesFromL1Tx(l1TxHash: string): Promise<string[]>
```

#### getMessageHashesFromL2Tx

```typescript
/**
  * Pulls all L2 => L1 message hashes out of an L2 transaction by hash.
  *
  * @param l2TxHash Hash of the L2 transaction to find messages for.
  * @returns List of message hashes emitted in the transaction.
  */
public async getMessageHashesFromL2Tx(l2TxHash: string): Promise<string[]> {
```

#### getL1TransactionReceipt

```typescript
/**
 * Finds the receipt of the L1 transaction that relayed a given L2 => L1 message hash.
 *
 * @param l2ToL1MsgHash Hash of the L2 => L1 message to find the receipt for.
 * @param pollForPending Whether or not to wait if the message hasn't been relayed yet.
 * @returns Receipt of the L1 transaction that relayed the message.
 */
public async getL1TransactionReceipt(
  l2ToL1MsgHash: string,
  pollForPending?: boolean
): Promise<TransactionReceipt>
```

#### getL2TransactionReceipt

```typescript
/**
  * Finds the receipt of the L2 transaction that relayed a given L1 => L2 message hash.
  *
  * @param l1ToL2MsgHash Hash of the L1 => L2 message to find the receipt for.
  * @param pollForPending Whether or not to wait if the message hasn't been relayed yet.
  * @returns Receipt of the L2 transaction that relayed the message.
  */
public async getL2TransactionReceipt(
  l1ToL2MsgHash: string,
  pollForPending?: boolean
): Promise<TransactionReceipt>
```

#### getMessageHashesFromTx

```typescript
/**
  * Generic function for looking for messages emitted by a transaction.
  *
  * @param layer Parameters for the network layer to look for a messages on.
  * @param txHash Transaction to look for message hashes in.
  * @returns List of message hashes emitted by the transaction.
  */
public async getMessageHashesFromTx(
  layer: Layer,
  txHash: string
): Promise<string[]>
```

#### getTransactionReceipt

```typescript
/**
 * Generic function for looking for the receipt of a transaction that relayed a given message.
 *
 * @param layer Parameters for the network layer to look for the transaction on.
 * @param msgHash Hash of the message to find the receipt for.
 * @param pollForPending Whether or not to wait if the message hasn't been relayed yet.
 * @returns Receipt of the transaction that relayed the message.
 */
public async getTransactionReceipt(
  layer: Layer,
  msgHash: string,
  pollForPending?: boolean
): Promise<TransactionReceipt>
```
