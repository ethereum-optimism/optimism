# Cross Domain Messaging

This specification covers the sending and relaying of messages, either from L2 to L1, or L1 to L2.

A high-level description I find useful to summarize the difference between the two flows is that:

1. From L2 to L1, messages are validated by verifying the inclusion of the message data in a mapping in a contract on the L2 state.
2. From L1 to L2, messages are validated simply by checking that the `ovmL1TXORIGIN` matches the expected address

## Cross Domain Messengers Contracts (aka xDMs)

There are two 'low level' bridge contracts (the L1 and L2 Cross Domain Messengers), which are 'paired' in the sense that they each other's addresses in order to validate the

## L2 to L1 messaging flow

**Starting on L2:**

- Any account on L2 may call `OVM_L2CrossDomainMessenger.sendMessage()` with the information for the L1 message (aka `xDomainCalldata`)
  - (ie. `_target`, `msg.sender`, `_message`)
  - This data is hashed with the `messageNonce` storage variable, and the hash is store in the `sentMessages` mapping (this is not actually used AFAIK)
  - The `messageNonce` is then incremented.
- The `OVM_L2CrossDomainMessenger` then passes the `xDomainCalldata` to `OVM_L2ToL1MessagePasser.passMessageToL1()`
  - the `xDomainCalldata` is hashed with `msg.sender` (ie. `ovmCaller`), and written to the `sentMessages` mapping.

**Then on L1:**

- The `Relayer` (and currently only the `Relayer`) may call `OVM_L1CrossDomainMessenger.relayMessage()` providing the raw message inputs and an L2 inclusion proof.
  - The relayer checks the following things:
    - in `_verifyStateRootProof()`:
      - checks that the fraud proof window has closed for the batch to which the transaction belongs
      - checks that the batch is stored in the `OVM_ChainStorageContainer`
    - in `_verifyStorageProof()`:
      - checks the proof to confirm that the message data provided is in the `OVM_L2ToL1MessagePasser.sentMessages` mapping
    - checks that this transaction has not already been written to the `successfulMessages` mapping.
  - The address of the L2 caller is then written to the `xDomainMessageSender` state var
  - the call is then executed
  - if it succeeds it is added to the `successfulMessages` and cannot be relayed again
  - regardless of success, an entry is written to the `relayedMessages` mapping

**Then the receiver (ie. `SynthetixBridgeToOptimism`):**

- Checks that the caller is the `OVM_L1CrossDomainMessenger` and that the `xDomainMessageSender` is the `synthetixBridgeToBase` on L2.

## L1 to L2 messaging flow

**Starting on L1:**

- Any account may call the L1xDM's `sendMessage()` function to submit their transaction data to the CTC's Transaction Queue, with the L2xDM as the `target`.

* (ie. `_target`, `msg.sender`, `_message`)
* This data is hashed with the `messageNonce` storage variable, and the hash is store in the `sentMessages` mapping (this is not actually used AFAIK)
* The `messageNonce` is then incremented.
* The `OVM_L2CrossDomainMessenger` then passes the `xDomainCalldata` to `OVM_L2ToL1MessagePasser.passMessageToL1()`
  - the `xDomainCalldata` is hashed with `msg.sender` (ie. `ovmCaller`), and written to the `sentMessages` mapping.

**Then on L1:**

- The `Relayer` (and only the `Relayer`) may call `OVM_L1CrossDomainMessenger.relayMessage()` providing the raw message inputs and an L2 inclusion proof.
  - The relayer checks the following things:
    - in `_verifyStateRootProof()`:
      - checks that the fraud proof window has closed for the batch to which the transaction belongs
      - checks that the batch is stored in the `OVM_ChainStorageContainer`
    - in `_verifyStorageProof()`:
      - checks the proof to confirm that the message data provided is in the `OVM_L2ToL1MessagePasser.sentMessages` mapping
    - checks that this transaction has not already been written to the `successfulMessages` mapping.
  - The address of the L2 caller is then written to the `xDomainMessageSender` state var
  - the call is then executed
  - if it succeeds it is added to the `successfulMessages` and cannot be relayed again
  - regardless of success, and entry is written to the `relayedMessages` mapping
