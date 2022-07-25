# L2 Output Root Proposals Specification

<!-- All glossary references in this file. -->

[g-rollup-node]: glossary.md#rollup-node
[g-mpt]: glossary.md#merkle-patricia-trie

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**

- [Proposing L2 Output Commitments](#proposing-l2-output-commitments)
- [L2 Output Commitment Construction](#l2-output-commitment-construction)
- [L2 Output Oracle Smart Contract](#l2-output-oracle-smart-contract)
- [Security Considerations](#security-considerations)
  - [L1 Reorgs](#l1-reorgs)
- [Summary of Definitions](#summary-of-definitions)
  - [Constants](#constants)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

After processing one or more blocks the outputs will need to be synchronized with L1 for trustless execution of
L2-to-L1 messaging, such as withdrawals. Outputs are hashed in a tree-structured form which minimizes the cost of
proving any piece of data captured by the outputs.
Proposers submit the output roots to L1 and can be contested with a fault proof,
with a bond at stake if the proof is wrong.

_Note_: Although fault proof construction and verification [is implemented in Cannon][cannon],
the fault proof game specification and integration of a output-root challenger into the [rollup-node][g-rollup-node]
are part of later specification milestones.

[cannon]: https://github.com/ethereum-optimism/cannon

## Proposing L2 Output Commitments

The proposer's role is to construct and submit output roots, which are commitments made on a configurable interval,
to the `L2OutputOracle` contract running on L2. It does this by running the [L2 output proposer](../op-proposer/),
a service which periodically queries the rollup node's
[`optimism_outputAtBlock` rpc method](./rollup-node.md#l2-output-rpc-method) for the latest output root derived
from the latest [finalized](rollup-node.md#finalization-guarantees) L1 block. The construction of this output root is
described [below](#l2-output-commitment-construction).

If there is no newly finalized output, the service continues querying until it receives one. It then submits this
output, and the appropriate timestamp, to the [L2 Output Root](#l2-output-root-smart-contract) contract's
`proposeL2Output()` function. The timestamp MUST be the next multiple of the `SUBMISSION_INTERVAL` value.

The proposer may also delete the most recent output root by calling the `deleteL2Output()` function.
The function can be called repeatedly if it is necessary to roll back the state further.

> **Note regarding future work:** In the initial version of the system, the proposer will be the same entity as the
> sequencer, which is a trusted role. In the future proposers will need to submit a bond in order to post L2 output
> roots, and some or all of this bond may be taken in the event of a faulty proposal.

## L2 Output Commitment Construction

The `output_root` is a 32 byte string, which is derived based on the a versioned scheme:

```pseudocode
output_root = keccak256(version_byte || payload)
```

where:

1. `version_byte` (`bytes32`) a simple version string which increments anytime the construction of the output root
   is changed.

2. `payload` (`bytes`) is a byte string of arbitrary length.

In the initial version of the output commitment construction, the version is `bytes32(0)`, and the payload is defined
as:

```pseudocode
payload = state_root || withdrawal_storage_root || latest_block_hash
```

where:

1. The `latest_block_hash` (`bytes32`) is the block hash for the latest L2 block.

1. The `state_root` (`bytes32`) is the Merkle-Patricia-Trie ([MPT][g-mpt]) root of all execution-layer accounts.
   This value is frequently used and thus elevated closer to the L2 output root, which removes the need to prove its
   inclusion in the pre-image of the `latest_block_hash`. This reduces the merkle proof depth and cost of accessing the
   L2 state root on L1.

1. The `withdrawal_storage_root` (`bytes32`) elevates the Merkle-Patricia-Trie ([MPT][g-mpt]) root of the [L2 Withdrawal
   contract](./withdrawals.md#the-l2tol1messagepasser-contract) storage. Instead of making an MPT proof for a withdrawal
   against the state root (proving first the storage root of the L2 withdrawal contract against the state root, then
   the withdrawal against that storage root), we can prove against the L2 withdrawal contract's storage root directly,
   thus reducing the verification cost of withdrawals on L1.

## L2 Output Oracle Smart Contract

L2 blocks are produced at a constant rate of `L2_BLOCK_TIME` (2 seconds).
A new L2 output MUST be appended to the chain once per `SUBMISSION_INTERVAL` which is based on a number of blocks.
The exact number is yet to be determined, and will depend on the design of the fault proving game.

The L2 Output Oracle contract implements the following interface:

```js
/**
 * @notice Accepts an L2 outputRoot and the timestamp of the corresponding L2 block. The
 * timestamp must be equal to the current value returned by `nextTimestamp()` in order to be
 * accepted.
 * This function may only be called by the Proposer.
 * @param _l2Output      The L2 output of the checkpoint block.
 * @param _l2BlockNumber The L2 block number that resulted in _l2Output.
 * @param _l1Blockhash   A block hash which must be included in the current chain.
 * @param _l1BlockNumber The block number with the specified block hash.
*/
  function proposeL2Output(
      bytes32 _l2Output,
      uint256 _l2BlockNumber,
      bytes32 _l1Blockhash,
      uint256 _l1BlockNumber
  )

/**
 * @notice Deletes the most recent output.
 * @param _l2Output The value of the most recent output. Used to prevent erroneously deleting
 *  the wrong root
 */
function deleteL2Output(bytes32 _l2Output) external

/**
 * @notice Computes the block number of the next L2 block that needs to be checkpointed.
 */
function nextBlockNumber() public view returns (uint256) {
```

## Security Considerations

### L1 Reorgs

If the L1 has a reorg after an output has been generated and submitted, the L2 state and correct output may change
leading to a faulty proposal. This is mitigated against by allowing the proposer to submit an
L1 block number and hash to the Output Oracle when appending a new output; in the event of a reorg, the block hash
will not match that of the block with that number and the call will revert.

## Summary of Definitions

### Constants

| Name                  | Value  | Unit    |
| --------------------- | ------ | ------- |
| `SUBMISSION_INTERVAL` | `1800` | seconds |
| `L2_BLOCK_TIME`       | `2`    | seconds |
