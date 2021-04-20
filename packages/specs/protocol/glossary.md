# Glossary

The following definitions are intended only to disambiguate some of the terms and abbreviations specific to Optimistic Ethereum protocol. They are intentionally kept incomplete, as they are described in more detail elsewhere within this specification.

## General concepts

- **Optimistic Rollup (ORU):** A design pattern using fraud proofs to enforce security assumptions on a layer 2 blockchain.
- **Optimistic Ethereum (OE):** Refers to the protocol described within this specification. An Optimistic Rollup implementation
- **Optimistic Virtual Machine (OVM):** A 'containerized' virtual machine designed run on the Ethereum Virtual Machine (EVM), and mirror the functionality and behavior of the EVM.
- **Domain:** A synchronous execution environment. Typically either the Ethereum Mainnet or Optimistic Ethereum.
- **Layer 1 (L1):** Typically refers to the Ethereum Mainnet. More generally the base chain which provides security to L2.
- **Layer 2 (L2):** Typically refers to the Optimistic Ethereum Rollup. More generally the chain which depends on L1 for security.

## OE System Components

- **Canonical Transaction Chain (CTC):** The chain of transactions executed on the Rollup chain.
- **State Commitment Chain (SCC):** The chain of state roots resulting from the execution of each transaction.
- **State Transitioner (ST):** During a fraud proof, this contract manages the setup of the prestate, execution of the transaction, and computation of the poststate.
- **Execution Manager (EM)**: The contract defining the OVM operations necessary to override EVM operations.
- **State Manager (SM):** The contract which manages a remapping of L2 to L1 addresses.
- **Fraud Prover:** The contract used to initiate and adjudicate a fraud proof.
- **Bond Manager:** The contract which accepts the collateral deposit required to act as a state proposer.
- **Cross Domain Messenger (xDM):** The pair of contracts used to pass messages between L1 and L2. These contracts are sometimes referred to as the L1xDM, and L2xDM.

## Implementation Specific Concepts

- **The Queue** / **Enqueued Transactions:** The Queue is an append only list of transaction on layer 1. Enqueued transactions must be added to the L2 chain within the Force Inclusion Period.
- **Force Inclusion Period:** The duration of time in which the Sequencer may still insert other transactions before an Enqueued Transaction is added to the CTC.
- **Stale transactions:** Enqueued transactions which have not yet been added to the Canonical Transaction Chain.
- **Batching:** The act of 'rolling up' or Merkleizing data for efficient on-chain storage, while making the data available in order to prove properties of the L2 state.
- **Safe/Unsafe opcodes:** Unsafe opcodes are those which would return a different value when executed on L1 or L2, and thus invalidate the property of deterministic execution.
- **Magic Strings:** The specific bytecode strings which are allowable despite containing unsafe opcodes, which force contracts to call to the Execution Manager.
- **Source** or **Feed:** The origin of a transaction, currently either the Sequencer, or Transaction Queue.
