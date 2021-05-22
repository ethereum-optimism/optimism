# Actor and Roles

## Sequencer

The Sequencer is a semi-privileged service provider in Optimistic Ethereum which enables instant transactions. The sequencer is given the role of assigning an order to L2 transactions, similarly to miners on L1.

There is only one sequencer at a time, allowing consensus on transactions to be reached extremely rapidly. Users can send transactions to the sequencer, and (within seconds) receive confirmation that their transaction was processed and will be included in the next rollup batch. These instant confirmations have weaker security than confirmed L1 transactions, but stronger security than 0-conf L1 transactions.

Once the sequencer's batch is confirmed on L1, the security is the same.

### On censorship resistance

In the event that a malicious Sequencer censors user's transactions, the user SHOULD `enqueue()` their transactions directly to the L1 Queue, forcing the sequencer to include them in the L2 chain within the `FORCE_INCLUSION_PERIOD`.

If any transaction in the queue are more than `FORCE_INCLUSION_PERIOD` blocks old, those transactions MUST be added to the Chain before the protocol will allow the Sequencer to add any other transactions.

In the event that the Sequencer stops submitting transactions entirely, the protocol will allow users to add transactions to the CTC by calling `appendQueueBatch()`

### Roles

- Receives new transactions from users
- SHOULD process transactions instantly to determine an optimal ordering
- Determines the ordering of transactions in the CTC, which MUST follow the [constraints](./processes/chains.md#properties-enforced-by-appendsequencerbatch) imposed by the `appendSequencerBatch()` function.
- SHOULD append transactions from the CTC's `queue` within the "Force Inclusion Period".

## Proposers

Proposers evaluate the transactions in the CTC, and 'commit' to the resulting state by writing them to the SCC. They must deposit a bond for the privilege of this role.
This bond will be slashed in the event of a successful fraud proof on a state root committed by the Proposer.

### Roles

- Process transactions from the CTC, and propose new state roots by posting them to the SCC.
- MUST be collateralized by depositing a bond to the `OVM_BondManager` .

**Future note:** The Proposer is currently identical to the Sequencer. This is expected to change.

## Verifier

Like Proposers, Verifiers evaluate the transactions in the CTC, in order to determine the resulting state root following each transaction.
If a Verifier finds that a proposed state root is incorrect, they can prove fraud, and earn a reward taken from the Proposer's bond.

### Roles

- Read transactions from the CTC, process them, and verify the correctness of state roots in the SCC.
- If an invalid state root is detected: initiate and complete a fraud proof.
  - Note that multiple accounts may contribute to a fraud proof and earn the reward.

## Users

Any account may transact on OE.

### Roles

- MAY post L2 transactions via the Sequencer's RPC endpoint, to be appended in sequencer batches
- MAY submit an L2 transaction via the CTC's queue on L1
  - Can be used to circumvent censorship by the Sequencer
  - Can be used to send a cross domain message from an L1 contract account.
