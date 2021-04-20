# Actor and Roles

## Sequencer

### Motivation

The Sequencer is a semi-privileged service provider in Optimistic Ethereum which enables instant transactions. The sequencer is given the role of assigning an order to L2 transactions, similarly to miners on L1.

There is only one sequencer at a time, allowing consensus on transactions to be reached extremely rapidly. Users can send transactions to the sequencer, and (within seconds) receive confirmation that their transaction was processed and will be included in the next rollup batch. These instant confirmations have weaker security than confirmed L1 transactions, but stronger security than 0-conf L1 transactions.

<!-- but stronger security than 0-conf L1 transactions. Why? -->

Once the sequencer's batch is confirmed on L1, the security is the same.

### Censorship resistance

In the event that a malicious Sequencer censors user's transactions, the user SHOULD `enqueue()` their transactions directly to the L1 Queue, forcing the sequencer to include them in the L2 chain within the `FORCE_INCLUSION_PERIOD`.

If any transaction in the queue are more than `FORCE_INCLUSION_PERIOD` blocks old, those transactions MUST be added to the Chain before the protocol will allow the Sequencer to add any other transactions.

In the event that the Sequencer stops submitting transactions entirely, the protocol will allow users to add transactions to the CTC by calling `appendQueueBatch()`

### Roles

- Receives new transactions from users
- SHOULD process transactions instantly to determine an optimal ordering
- Determines the ordering of transactions in the CTC, which MUST follow the [constraints](./processes/chains.md#properties-enforced-by-appendsequencerbatch) imposed by the `appendSequencerBatch()` function.
- SHOULD append transactions from the CTC's `queue` within the "Force Inclusion Period".

## Proposers

**Roles:**

- Process transactions from the CTC, and propose new state roots by posting them to the SCC.
- MUST be collateralized by depositing a bond to the `OVM_BondManager` .

## Verifier

**Roles:**

- Read transactions from the CTC, process them, and verify the correctness of state roots in the SCC.
- If an invalid state root is detected: initiate and complete a fraud proof.
  - Note that multiple accounts may contribute to a fraud proof and earn the reward.

## Users

**Roles:**

- MAY post L2 transactions via the Sequencer's RPC endpoint, to be appended in sequencer batches
- MAY submit an L2 transaction via the CTC's queue on L1
  - Can be used to circumvent censorship by the Sequencer
  - Can be used to send a cross domain message from an L1 contract account.
