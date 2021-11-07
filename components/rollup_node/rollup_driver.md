# Rollup Driver

The Rollup Driver is the core code which guarantees that the canonical L1 state is accurately reflected in the EE. It does this by connecting to an L1 node, and using the pure function which maps L1 blocks to L2 block inputs as defined by [block generation][block-gen]. Once it knows the block inputs, it checks the EE to make sure that the blocks there all have the correct inputs.

This is a functionality that is shared between all configurations of the rollup node, including the sequencer. Because block generation defines the L1 block -> L2 block input transformation as a pure function, it can be completely stateless and has no DB.

## Basic Functionality

We define the `latestL1ConfirmedL2Block` as the latest L2 block in the EE which is marked as L1-confirmed. This can be found by starting at the head of the L2 EE, and iterating backwards until we find the first L2 block which is marked as L1-confirmed.

The core logic which the CL executes is as follows.

1. Find `latestL1ConfirmedL2Block`. Then get `latestL1ConfirmedL2Block.l1BlockNumber` and `latestL1ConfirmedL2Block.l1BlockHash` from the EE. Compare these to L1 with an RPC query to the L1 node's RPC.
    - if they DO match, continue to 2.
    - if they DO NOT match, then there has been an L1 reorg. In that case:
        - Signal the reorg.
        - Iterate backwards until we find the latest L2 block whose `l1BlockNumber` and `l1BlockHash` DO match the L1 node RPC.
        - Set the EE's head to this L2 block (thereby updating `latestL1ConfirmedL2Block`).
2. Iterate forward through L1 blocks, starting from `latestL1ConfirmedL2Block.l1BlockNumber + 1`. For each:
    - Generate the L2 block inputs for this L1 block as defined by [block generation][block-gen]. For each L2 block input:
        - If the L2 block DOES NOT exist in the EE already, insert it and update the EE head.
        - If the L2 block DOES exist in the EE already (as sequencer confirmed), then compare the generated block input to the existing block input.
            - If they DO match, mark the L2 block as L1 confirmed.
            - If they DO NOT match, there has been an L2 reorg. Signal the reorg, insert the correct block, and update the EE head.
3. Wait and GOTO 1.

[block-gen]: ./components/rollup_node/block_gen.md