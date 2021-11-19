# Glossary of Optimistic Ethereum Terms

### Layer 1 (L1)

An Ethereum network with its own native security mechanism (ie. proof of work, proof of stake or proof of authority).

### Layer 2 (L2)

A network that derives its security from a layer one.

### L2 State Root Oracle

A smart contract on L1, providing access to historical L1 block hashes. The existence of this contract makes it possible to prove the existence of a particular L2 state root within a previous L1 block.

### Block Generation (AKA Block Derivation)

All of the data necessary to reconstruct the history and state of L2 is contained within L1 blocks. Block Generation is the process of converting data contained in L1 blocks into L2 blocks.

### Epoch

A continuous sequence of 0 or more L2 blocks. There is a one to one correspondence between L2 Epochs, and L1 blocks.

**Example usage:** "Epoch 20 corresponds to L1 block 637, epoch 21 corresponds to block 638, etc".

### Feeds

Feeds are append only lists, each of which is contained in a smart contract on Layer 1. The state of the Rollup Chain is determined by the data written to these contracts.

### Sequencer Feed

The Sequencer Feed is a contract which contains the data representing sequencer submitted blocks, each element in this feed will be used to generate a "sequencer block".

### Deposit Feed

The Deposit Feed is a contract which contains deposit data. Each element in this feed will be used to generate a "deposit block".

### Sequencing Window

The number of L1 blocks during which the Sequencer _may_ insert Sequencer blocks before a Deposit block.

If the Sequencer Window is 2 blocks, and Alice makes a Deposit in Block 420, her deposit WILL be included (by the Block Generation process) at block 422.

### Execution Environment (EE)

The virtual machine in which transactions are applied to the pre-state to generate the post-state.

### Rollup Node AKA Rollup Driver

The core client software which reads data from an L1 node, and transforms it into L2 data. Both the Sequencer, and Verifier nodes run a Rollup Node, albeit in different configurations.

### Consensus Layer

By analogy to the architecture of ETH2 we can also describe Layer 1 as the "Consensus Layer", as it defines the canonical history and state of the Layer 2 chain.

### Sequencer

The privileged entity which accepts L2 transactions from users, and determines the ordering of those transactions with respect to each other and deposit transactions.


### Verifier

A verifier is any Rollup Node operator who is not the Sequencer. Verifiers compute the state of the L2 chain, and compare it to the states proposed by the Sequencer.

### Deposit transaction (via the Deposit Feed)

A transaction which originates on L1, by way of a message sent to the Deposit Feed contract. This message can be sent either by a contract or an externally owned account.

### Sequencer transaction (via the Sequencer feed)

A transaction which is sent directly to the sequencer from user.

### L2 State Oracle (AKA L2 Block Oracle)

The on-chain component to which the Sequencer writes the L2 state roots.
The L2 State oracle also implements the k-section game, and single step verifier.

### Single Step Verifier

An on-chain

### Dispute game

The Dispute Game is the process by which a verifier


### Preimage
### Pure function


