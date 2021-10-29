# Optimistic Ethereum Overview
*// Note: this section assumes you've read the [introduction](./introduction.md)!*
## Arcitecture Design Goals

- Execution-Level EVM Equivalence
  - The developer experience should be indentical to L1 except where L2 introduces a fundamental difference.
    - No special compiler
    - No unexpected gas costs
    - Out-of-the-box TX traces
    - All tooling "just works"--configure chain ID, good to go
  - Fundamental L2 differences are limited to the bare minimum, still use existing L1 standards, and require only block-level modifications. These include:
    - Fee logic which accounts for L1 calldata costs (still uses existing L1 tx type)
    - A new transaction type for deposits (still uses L1 tx envelope schema)
    - EIP1559 algorithm that accounts for L2 blocktimes
- Maximal compatibility with Eth1 nodes
  - Uses the ETH2 merge's Engine API for:
    - Fork-choice: change head based on L1 reorgs of data (and leverage undisputed state-roots for finalized checkpoint)
    - Block-production: sequencing blocks with any Eth1 node
    - Block-insertion: Eth1 node has specialized EVM and DB, valuable for L2, instead of reinventing the wheel
  - Leverages sync: verify block inputs based on L1, but do state-sync via L2 p2p network
  - Leverage tx propagation: can use existing mempool infrastructure
- Simple, stateless services
  - Whenever possible (e.g. in batch submitter and synchronizer), services contributing to the rollup infrastructure do not persist state
  - Given an L1 node, and an L2 execution engine, all other services can recover to full operation when started with a fresh DB

## Components

![Architecture Diagram](./assets/architecture.svg)

### L1 Components
- **Feeds**: ("Data availability layer"): A **feed** is an append-only log of data (e.g. of deposits, or sequencer batches) which must guarantee that:
  - All entries are deterministically indexable by off-chain parties.
  - The fraud proof VM can find any feed entry in a bounded number of steps, such that the off-chain indexing is sufficient to generate a witness for verifing any individual step on-chain.
  - Note: a "feed" may not necessarily correspond to an on-chain contract (e.g. which stores a hash of each entry). Instead, a feed refers to the actual log of data which is derivable from L1, which may or may not use a contract. For more info, check out [this discussion.](https://github.com/ethereum-optimism/optimistic-specs/issues/14)
- **L2 Block Oracle**: A cryptoeconomic light client of the L2. Will finalize hashes of the L2 state on L1, once the dispute period has passed. Used to validate withdrawals. *(Note: contract separation is not yet final)*
  - **Proposal Manager**: Handles conflicting state proposals to determine malicious party
    - Maintains a set of ongoing optimistic proposals of the L2 state
    - Ensures that state proposers are sufficiently bonded
    - Distributes bond payouts
  - **k-section game**
    - executed to narrow in on the first disagreed-upon step between proposers
    - played between two parties ("all people who agree with a proposal" may be considered a single party)
    - merkle tree of execution steps between the trusted start and disputed end
  - **Single Step Verifier** for executing the earliest disagreed-upon step
    - Loads witness data:
      - step witness parts can be represented as `generalized index -> bytes32`,
        to simplify interaction with the partial structure (no tree management, every lookup is a simple math evaluation to get the right key)
      - involved contract code entries can be large, TBD if put in storage or better of in call-data
      - MPT nodes are just a dict (of internal MPT node hash to MPT node content)
    - Runs verification step
    - Registers result in bond-manager (slash sequencer, reward challenger, or vice-versa if spam attempt)
- **Cross-domain Messenger**: Message-passing abstraction contract used by applications developers (e.g. token bridge implementations)
    - Sends messages into L2, reads messages out from L1
    - Log authorized calls as forced L2 transactions (deposits)
    - Verify L2 data statelessly with MPT proof to undisputed state-root (withdrawals)

### L2 Components

- **Rollup Node** (previously "Data Transport Layer"):
  - Sync data from the L1: call payload-insertion method of Engine API
  - Track L1 data tip, and track state-roots: call forkchoice method of Engine API to finalize (undisputed state root) and reorg (follow head derived from L1)
  - Create execution payloads:
    - L2 Blocks as a whole from the L1 (i.e. include header data for batch of transactions) (TBD)
    - Follow rollup rules to batch remaining individual transactions
    - L1 may log special transactions, created by the Gateway,
      which are paid for (i.e. signed by dummy key, and sent to L2 Gateway receiver contract, avoid new transaction type)
  - Verifier (if loaded with a funded L1 account to bond the challenge):
    - Block is executed with engine API, resulting state root is cached
    - State roots are synced (maybe with a slight delay)
    - If roots are different, run the Fraud-Proof Generator and start challenger agent job
- **Execution Engine** (previously "Eth1 client")
  - Serves new [Engine API](https://hackmd.io/@n0ble/consensus_api_design_space) (Work in progress, introduced in The Merge hardfork)
  - Maintains the L2 state (just like it would on L1)
  - Syncs state to other L2 nodes for fast onboarding. Engine API can inform of finalized checkpoint,
    to efficiently sync to trusted point, before processing block by block to the head of the L2 chain.
- **Witness Generator**:
  - Highly experimental concept implementation (work in progress!): [Macula](https://github.com/protolambda/macula)
  - Python: readability, not performance critical, just a temporary sub-process
  - Requires bare minimum API methods from the Engine:
    - get a MPT node (actually already present in p2p of eth1)
    - get contract-code by contract-code hash
  - Inputs:
    - the API address to call for MPT node data and contract code
    - the last 256 block-hashes
    - the failing execution payload
  - Output: fraud proof
    - array of `bytes32`, root of each step, in sequence. (can be thousands)
    - array of `AccessData`, encoding the set of generalized indices, code-hashes and MPT-node-hashes that were accessed this step.
      as well as for the step-by-index (optimization, see generator spec).
    - dict of `bytes32->(bytes32, bytes32)`: DB of all tree structure of the steps, without duplicating anything
    - dict of `bytes->bytes`: map MPT node key (not leaf key, the actual internal node hash referenced in parent MPT nodes) to MPT node (RLP encoded path and child node data)
    - dict of `bytes32->bytes`: map contract-code-hash to full contract code
    - metadata of fraud (when/where/etc.)
- **Challenger Agent**:
  - Take generated fraud proof (e.g. JSON file)
  - Submit bond: control spam, challenges get back the bond, if they are actually valid.
  - Run dispute game, retrieve disputed step index (WIP implementation [here](https://github.com/statechannels/dispute-game))
    - every step before this is agreed on by both parties
    - the first differing step will be verified by reproducing the correct outcome on-chain
  - Build witness data:
     1. get root of the disputed step
     2. get generalized indices
     3. load nodes as identified by the AccessData from shared dicts
     4. format witness data as L1 transaction input
  - Sign tx, finish dispute (resubmit if not confirmed in time)
- **Batch Submitter**:
  - Finalizes pre-confirmed sequencer transactions onto L1
  - Writes to sequencer feed

