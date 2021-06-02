# @eth-optimism/message-relayer

This package contains:

1. A service for relaying messages from L2 to L1.
2. Utilities for finding these messages and relaying them.

## Installation

```
yarn add @eth-optimism/message-relayer
```

## Relay Utilities

### getMessagesAndProofsForL2Transaction

Finds all L2 => L1 messages sent in a given L2 transaction and generates proof for each.

#### Usage

```typescript
import { getMessagesAndProofsForL2Transaction } from '@eth-optimism/message-relayer'

const main = async () => {
  const l1RpcProviderUrl = 'https://layer1.endpoint'
  const l2RpcProviderUrl = 'https://layer2.endpoint'
  const l1StateCommitmentChainAddress = 'address of OVM_StateCommitmentChain from deployments page'
  const l2CrossDomainMessengerAddress = 'address of OVM_L2CrossDomainMessenger from deployments page'
  const l2TransactionHash = 'hash of the transaction with messages to relay'

  const messagePairs = await getMessagesAndProofsForL2Transaction(
    l1RpcProviderUrl,
    l2RpcProviderUrl,
    l1StateCommitmentChainAddress,
    l2CrossDomainMessengerAddress,
    l2TransactionHash
  )

  console.log(messagePairs)
  // Will log something along the lines of:
  // [
  //   {
  //     message: {
  //       target: '0x...',
  //       sender: '0x...',
  //       message: '0x...',
  //       messageNonce: 1234...
  //     },
  //     proof: {
  //       // complicated
  //     }
  //   }
  // ]

  // You can then do something along the lines of:
  // for (const { message, proof } of messagePairs) {
  //   await l1CrossDomainMessenger.relayMessage(
  //     message.target,
  //     message.sender,
  //     message.message,
  //     message.messageNonce,
  //     proof
  //   )
  // }
} 

main()
```
