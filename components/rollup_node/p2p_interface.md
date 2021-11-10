# P2P Interface

Previously known as the "sequencer replicator", Work In progress.

Design direction:
- distribute workload of L2 sequencer
- create redundancy
- permissionless access to head of the L2 chain
- access to unconfirmed data in user RPC
- mirror forkchoice boundaries separation in user RPC, with new post-merge standard:
  - finalized == undisputed
  - safe (previously known as "latest") = confirmed data on L1, might be disputed or reorged via L1, but safe w.r.t. L2.
  - unsafe = unconfirmed, pending on L1, but known via P2P on L2 ahead of time.

----



The sequencer replicator connects directly to the sequencer. It requests pending L2 blocks which have not yet been batch submitted,
and inserts them into the tip of the local EE, marked only as sequencer confirmed (NOT L1 confirmed). The sequencer replicator is connected to the sequencer EE via RPC.

A rollup node using the replicator will place additional trust in the sequencer's local state being resolved to L1 via batch submission. However, it will still execute all blocks itself, so that the EE state will always be something that *is possible* in the future, even if it doesn't end up manifesting that way on L1.

## Basic Functionality

- The sequencer replicator may only modify or add state which is at the end of the EE, and not yet L1 confirmed.
  - The replicator is **NEVER** to modify any blocks in the EE which are marked L1 confirmed.
  - ONLY sequencer confirmed blocks may written or overwritten by the replicator.
- Halted mode: If a reorg is detected by the [consensus layer][consensus-layer], or if the sequencer sends conflicting data (outlined below), the replicator service should halt. It should do this either for a configured time period, until manual intervention by the node operator, or perhaps until the CL indicates that it has finished syncing. If the replicator is halted, then the Replicating Verifier will subsequently behave identically to the L1 verifier.

Core logic:

1. Get the local EE head, and query the corresponding sequencer block via RPC.
  - if the blocks DO NOT agree, something is up with the sequencer. Set the head to `latestL1ConfirmedL2Block`, and halt.
2. Query the sequencer's RPC for the blocks between the local EE head and the sequencer head. For each returned block:
  - Get the block input and apply it to the head. (As sequencer confirmed--even if sequencer says L1 confirmed.)
  - Make sure the resulting local blockhash and stateroot match what sequencer returned. If they DO NOT agree, something is up with the sequencer. Set the head to `latestL1ConfirmedL2Block`, and halt.
3. Wait and GOTO 1.

[consensus-layer]: ./consensus_layer.md