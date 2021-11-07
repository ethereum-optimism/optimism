# Rollup Node

The consensus module of Optimistic Ethereum.

## Summary

The Rollup Node is a consensus client that determines the latest state of the rollup. It reads state from L1, and possibly the sequencer, to compute the L2 state. The [block generation spec][block-gen] defines the rules by which L1 state is converted into L2 blocks.

## Types of Rollup Nodes
There are two primary classes of rollup nodes, corresponding to different configurations of the software stack.

1. **Sequencer**: Accepts user-sent and deposit transactions, orders them into L2 blocks, and submits them to L1 in batches. Provides RPC access to pending L2 blocks in advance of L1 submission.
2. **Verifier**: Watches L1 feeds, reconstructs L2 block inputs, and inserts them into the EE to determine the canonical L2 state. May also track pending sequencer state, in some configurations.

### Subcomponents Used for Each Node Type
Of the components outlined below, different node types use different combinations of them, either manually or optionally.

| Node Type \ Component | Execution Engine | Rollup Driver | Sequencer Replicator | Block Producer | Batch Submitter |
|-----------------------|------------------|----------------------|----------------------|----------------|-----------------|
| Verifier  | X                | X                    | optional                    |                |                 |
| Sequencer             | X                | X                    |                      | X              | X               |

## Components

### [L2 Execution Engine][exec-engine]

The execution engine implements the [execution specification][execution-spec].  Most L1 clients will soon be converted into execution engines. The rollup client will communicate to the engine via a JSON-RPC interface ([WIP][execution-engine-rpc]).

One of the main goal of the rollup client is to use the execution engine without modification.

### [Rollup Driver][rollup-driver]

The rollup driver connects to an L1 node, and tracks the feed data comprising of deposits and sequencer batches. Based on the rules of [block generation][block-gen], it is able to compute sets ("epochs") of L2 blocks as a stateless, pure function of ranges of L1 blocks.

While connected to the Execution Engine, it uses the Engine API to progress the L2 state as new data comes in from L1. The rollup driver will reorganize the L2 state if it detects a difference between the EE and what's on L1, and provides a way for other services to subscribe to reorgs.

### [Sequencer Replicator][sequencer-replicator]

The sequencer replicator connects directly to the sequencer. It requests pending L2 blocks which have not yet been batch submitted, and inserts them into the tip of the local EE.

These blocks are marked in the EE as not yet confirmed on L1. Replicating verifiers still run a CL, which will later mark them as L1-confirmed within the EE.

### [Block Producer][block-producer]

The block producer service is run by the sequencer to progress the L2 state in advance of batch submission. The block producer repeatedly calls `engine_preparePayload` to get a new L2 block and then uses `engine_executePayload` to insert the resulting block into the state. When new L1 blocks appear, it passes the corresponding deposit transactions to `engine_preparePayload`.

### [Batch Submitter][batch-submitter]

The batch submitter takes the pending blocks which have been inserted into the sequencer's L2 state by the block producer, and sends corresponding transactions in batches to L1.

The batch submitter encodes these transactions based on the rules of [block generation][block-gen], effectively as the reverse of the process carried out by the rollup driver. As a result:
- Blocks previously inserted by the Block Producer into the Sequencer's EE will now appear in L1 Verifiers' EEs.
- Blocks marked as sequencer-confirmed in both the Sequencer and in Replicating verifiers will now be marked as L1-confirmed.

[execution-spec]: https://github.com/ethereum/execution-specs
[execution-engine-rpc]: https://hackmd.io/@n0ble/consensus_api_design_space
[block-gen]: ./components/rollup_node/block_gen.md
[exec-engine]: ./exec_engine.md
[rollup-driver]: ./consensus_layer.md
[sequencer-replicator]: ./sequencer_replicator.md
[block-producer]: ./block_producer.md
[batch-submitter]: ./batch_submitter.md