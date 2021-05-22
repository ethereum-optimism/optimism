# Cross Domain Messaging

This specification covers the sending and relaying of messages, either from L2 to L1, or L1 to L2.

A high-level description I find useful to summarize the difference between the two flows is that:

1. From L2 to L1, messages are validated by verifying the inclusion of the message data in a mapping in a contract on the L2 state.
2. From L1 to L2, messages are validated simply by checking that the `ovmL1TXORIGIN` matches the expected address

## Cross Domain Messengers Contracts (aka xDMs)

There are two 'low level' bridge contracts (the L1 and L2 Cross Domain Messengers), which are 'paired' in the sense that they reference each other's addresses in order to validate cross domain messages.

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
- The validity of the message is confirmed by the following functions:
  - `_verifyStateRootProof()`:
    - checks that the fraud proof window has closed for the batch to which the transaction belongs.
    - checks that the batch is stored in the `OVM_ChainStorageContainer`.
  - `_verifyStorageProof()`:
    - checks the proof to confirm that the message data provided is in the `OVM_L2ToL1MessagePasser.sentMessages` mapping
    - checks that this transaction has not already been written to the `successfulMessages` mapping.
- The address of the L2 `ovmCALLER` is then written to the `xDomainMessageSender` state variable
  - the call is then executed, allow the `target` to query the value of the `OVM_L1CrossDomainMessenger.xDomainMessageSender` for authorization.
- if it succeeds it is added to the `successfulMessages` and cannot be relayed again.
- regardless of success, an entry is written to the `relayedMessages` mapping.

**Then the receiver (ie. `SynthetixBridgeToOptimism`):**

- Checks that the caller is the `OVM_L1CrossDomainMessenger` and that the `xDomainMessageSender` is the `synthetixBridgeToBase` on L2.

## L1 to L2 messaging flow

**Starting on L1:**

- Any account may call the L1xDM's `sendMessage()`, specifying the details of the call that the L2xDM should make.
- The L1xDM call `enqueue` on the CTC to add to the Transaction Queue, with the L2xDM as the `target`.
  - The [`Transaction.data`](../data-structures.md#transaction) field should be ABI encoded to call `OVM_L2CrossDomainMessenger.relayMessage()`.

**Then on L2:**

- A transaction will be sent to the `OVM_L2CrossDomainMessenger`.
- The cross-domain message is deemed valid if the `ovmL1TXORIGIN` is the `OVM_L1CrossDomainMessenger`.
  - If not valid, execution reverts.
- If the message is valid, the arguments are ABI encoded and keccak256 hashed to `xDomainCalldataHash`.
- The `succesfulMessages` mapping is checked to verify that `xDomainCalldataHash` has not already been executed successfully.
  - If an entry is found in `succesfulMessages` execution reverts.
- A check is done to disallow calls to the `OVM_L2ToL1MessagePasser`, which would allow an attacker to spoof a withdrawal.
  - Execution reverts if the check fails.
  - **Future note:** The `OVM_L2ToL1MessagePasser`, and this check should be removed, in favor of putting the `sentMessages` mapping into the L2xDM.
- The address of the L2 `ovmCALLER` is then written to the `xDomainMessageSender` state variable
  - the call is then executed, allow the `target` to query the value of the `OVM_L1CrossDomainMessenger.xDomainMessageSender` for authorization.
- If it succeeds it is added to the `successfulMessages`.
