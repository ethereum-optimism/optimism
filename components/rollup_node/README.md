# Rollup Node

The consensus-layer of Optimism.

This design mirrors the separation of L1 consensus- and execution-layer to L2,
for maximum compatibility and minimal complexity.

## Summary

The Rollup Node is a consensus client that tracks the rollup:
- Fork-choice: following L1, a reorg on L1 results in a reorg of L2
- Input Data: deposits and sequencer data extracted from L1
- Latest blocks: propagate optimistically via p2p interface, overrideable by L1

The input data and forkchoice information collected from L1 (and optimistically via p2p)
are then forwarded to the execution-layer, also known as "execution engine",
via the standard [Engine API][engine-api].

The design is minimalist:
- The rollup consensus-layer is largely derived from L1 properties, enabling a simple and stateless L2 consensus.
- The rollup execution-layer is largely shared with L1, equivalent in all ways except the addition of rollup-deposits.

## Types of Rollup Nodes

There are two primary classes of rollup nodes, corresponding to different configurations of the software stack.

1. **Sequencer**: Accepts user-sent and deposit transactions, orders them into L2 blocks, and [submits them in batches][batch-submitter] to L1.
2. **Verifier**: Watches L1 feeds, [reconstructs L2 blocks from inputs][block-gen], and inserts them into the EE to determine the canonical L2 state.

**Both** update their coupled execution-engine with L2 blocks and fork-choice instructions, to maintain their view of the rollup state.

Optionally the sequencer(s) may share the L2 blocks via the L2 p2p network before the L1 can confirm them,
which verifiers can optionally use to sync to the very tip (marked "unsafe" in JSON-RPC) of the rollup.
This enables permissionless L1-like latency for users, but not canonical until available on the L1.

## Services

### L2 Execution Engine

The [Execution Engine][exec-engine] (EE) is *separated from the rollup node*, following the separated consensus/execution layer design of L1.

Execution tasks of the engine include EVM execution, transaction-pool and state sync.
The rollup node communicates to the engine via the [Engine JSON-RPC interface][engine-api]
(Secured and separated from the regular user JSON-RPC).

Most L1 clients are working on conversion to execution engines, as part of the [The Merge][the-merge].

Every execution-engine that is built for L1 can be used, with a minimal set of changes:
- Support a Deposit-transaction type: clean and efficient separation from regular transactions, following [EIP 2718][EIP-2718], similar to EIP-1559 transactions.
- Enable the consensus-layer to specify transactions during payload preparation: L2 blocks can then be formed on just the inputs, and deposits can be processed.
- Adjustment to Fee metering

All three of these features overlap with forthcoming L1 updates and may be unified at a later stage:
Validator withdrawals, system-transactions, and time-aware base fee calculation [EIP 4396][EIP-4396].

### Rollup Driver

The [Rollup Driver][rollup-driver] is the main service *within the rollup node*, mapping L1 data to L2 chain progress.

Based on the rules of [block generation][block-gen], it is able to compute sets ("epochs") of L2 blocks as a stateless, pure function of ranges of L1 blocks.

While connected to the Execution Engine, it uses the Engine API to progress the L2 state as new data comes in from L1.
The rollup driver will instruct the engine to reorganize the L2 state if it detects a difference between the local chain and remote chain as confirmed on L1,
and provides a way for other services to subscribe to reorgs.

### Block Producer

The [Block Producer][block-producer] is an optional service to run as sequencer, *within the rollup node*,
to progress the L2 state in advance of batch submission.

When tasked to produce a new rollup block, the rollup node adds `PayloadAttributesV1` to the `engine_forkchoiceUpdated`
call (see [Engine api][engine-API]) to instruct the engine to start sequencing a block of pending contents from the transaction pool.

This execution block is then retrieved with `engine_getPayload`, to then be pushed to L1 for availability by the
[Batch submitter][batch-submitter] (optionally packed together with other L2 blocks),
and optionally directly the L2 network for faster tip updates.

### Batch Submitter

The [Batch Submitter][batch-submitter] service takes a queue of locally produced payloads,
and submits the corresponding transactions in batches to L1.

The batch submitter encodes these transactions based on the rules of [block derivation][block-derivation], effectively as the reverse of the process carried out by the rollup driver. As a result:
- Blocks previously inserted by the Block Producer into the Sequencer's EE will now appear in L1 Verifiers' EEs.
- Blocks marked as sequencer-confirmed in both the Sequencer and in Replicating verifiers will now be recognized as L1-confirmed.

### P2P Interface

The [P2P Interface][p2p-interface] service offers an additional method of syncing the latest L2 blocks,
faster than the L1 can make them available.

This is implemented with an optimistic process:
1. Sequencer publishes the L2 block through a lightweight L2 P2P network (single gossip topic)
2. Verifier listens on the P2P network, verifies sequencer signature, and applies the block to their L2 EE.
3. If the data cannot be confirmed on L1, or if conflicting data is confirmed, then the bad L2 block is reorganized away.

This service is optional: while the worst-case data-retrieval is covered by L1 in the same way, the happy-path improves:
- Verifiers can stay in sync with the tip of the rollup with lower latency
- Sequencers can distribute the server work

The amount of L2 blocks that can be optimistically applied to the L2 EE is constrained
by the supported reorg-depth by the execution engine, to preserve the ability to process reorgs of the L2 by L1:
ultimately the L1 chain determines the canonical rollup chain.


[engine-api]: https://github.com/ethereum/execution-apis/tree/main/src/engine
[the-merge]: https://eips.ethereum.org/EIPS/eip-3675
[EIP-2718]: https://eips.ethereum.org/EIPS/eip-2718
[EIP-4396]: https://eips.ethereum.org/EIPS/eip-4396
[block-gen]: ./block_gen.md
[exec-engine]: ./exec_engine.md
[rollup-driver]: ./consensus_layer.md
[block-producer]: ./block_producer.md
[batch-submitter]: ./batch_submitter.md
[p2p-interface]: ./p2p_interface.md
