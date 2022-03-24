# L2 Output Root Proposals Specification

<!-- All glossary references in this file. -->
[g-rollup-node]: glossary.md#rollup-node

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**

- [Definitions](#definitions)
  - [Constants](#constants)
  - [Types](#types)
- [Proposing L2 Output Commitments](#proposing-l2-output-commitments)
- [L2 Output Commitment Construction](#l2-output-commitment-construction)
- [L2 Output Oracle Smart Contract](#l2-output-oracle-smart-contract)
- [Security Considerations](#security-considerations)
  - [L1 Reorgs](#l1-reorgs)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

After processing one or more blocks the outputs will need to be synchronized with L1 for trustless execution of
L2-to-L1 messaging, such as withdrawals. Outputs are hashed in a tree-structured form which minimizes the cost of
proving any piece of data captured by the outputs.
Proposers submit the output roots to L1 and can be contested with a fault proof,
with a bond at stake if the proof is wrong.

*Note*: Although fault proof construction and verification [is implemented in Cannon][cannon],
the fault proof game specification and integration of a output-root challenger into the [rollup-node][g-rollup-node]
are part of later specification milestones.

[cannon]: https://github.com/ethereum-optimism/cannon

## Definitions

### Constants

| Name                   | Value | Unit    |
| ---------------------- | ----- | ------- |
| `SUBMISSION_INTERVAL` | `1800` | seconds |
| `L2_BLOCK_TIME`        | `2`   | seconds |

### Types

The `ForkSpec` type contains the height and blockhash for a block in the L1 chain.

```js
struct ForkSpec {
    uint256 blockHeight;
    bytes32 blockHash;
}
```

## Proposing L2 Output Commitments

The proposer's role is to construct and submit output commitments on a configurable interval to a contract on , which
it does by running the [L2 output submitter](../l2os/). This service periodically queries the rollup
 node's [`optimism_outputAtBlock` rpc method](./rollup-node.md#l2-output-rpc-method) for the latest output root derived
 from the latest [finalized](rollup-node.md#finalization-guarantees) L1 block. The construction of this output root is
 described [below](#l2-output-commitment-construction).

If there is no newly finalized output, the service continues querying until it receives one. It then submits this
output, and the appropriate timestamp, to the [L2 Output Commitment](#l2-output-commitment-smart-contract) contract's
`appendL2Output()` function. The timestamp MUST be the next multiple of the `SUBMISSION_INTERVAL` value.

## L2 Output Commitment Construction

This merkle-structure is defined with [SSZ], a type system for merkleization and serialization, used in
L1 (beacon-chain). However, we replace `sha256` with `keccak256` to save gas costs in the EVM.

[SSZ]: https://github.com/ethereum/consensus-specs/blob/dev/ssz/simple-serialize.md

```python
class L2Output(Container):
  state_root: Bytes32
  withdrawal_storage_root: Bytes32  # TODO: withdrawals specification work-in-progress
  latest_block: ExecutionPayload  # includes block hash
  history_accumulator_root: Bytes32  # Not functional yet
  extension: Bytes32
```

The `state_root` is the Merkle-Patricia-Trie ([MPT][g-mpt]) root of all execution-layer accounts,
also found in `latest_block.state_root`: this field is frequently used and thus elevated closer to the L2 output root,
as opposed to retrieving it from the pre-image of the block in `latest_block`,
reducing the merkle proof depth and thus the cost of usage.

The `withdrawal_storage_root` elevates the Merkle-Patricia-Trie ([MPT][g-mpt]) root of L2 Withdrawal contract storage.
Instead of a MPT proof to the Withdrawal contract account in the account trie,
one can directly access the MPT storage trie root, thus reducing the verification cost of withdrawals on L1.

The `latest_block` is an execution-layer block of L2, represented as the [`ExecutionPayload`][ExecutionPayload] SSZ type
defined in L1. There may be multiple blocks per L2 output root, only the latest is presented.

[ExecutionPayload]: https://github.com/ethereum/consensus-specs/blob/dev/specs/bellatrix/beacon-chain.md#executionpayload

The `history_accumulator_root` is a reserved field, elevating a storage variable of the L2 chain that maintains
the [SSZ] merkle root of an append-only `List[Bytes32, MAX_ITEM_COUNT]` (`keccak256` [SSZ] hash-tree-root),
where each item is defined as `keccak256(l2_block_hash ++ l2_state_root)`, one per block of the L2 chain.
While reserved, a zeroed `Bytes32` is used instead.
This is a work-in-progress, see [issue 181](https://github.com/ethereum-optimism/optimistic-specs/issues/181).
`MAX_ITEM_COUNT` and/or other parameters will be defined in the withdrawals milestone.

The `extension` is a zeroed `Bytes32`, to be substituted with a SSZ container to extend merkleized information in future
upgrades. This keeps the static merkle structure forwards-compatible.

## L2 Output Oracle Smart Contract

L2 blocks are produced at a constant rate of `L2_BLOCK_TIME` (2 seconds).
A new L2 output MUST be appended to the chain once per `SUBMISSION_INTERVAL` (1800 seconds). Note that this interval is\
based on L2 time. It is OK to have L2 outputs submitted at larger or small intervals.

The L2 Output Oracle contract implements the following interface:

```js
/**
 * Data necessary to ensure that the output is being written to the expected fork of the L1 chain.
 * This protects against an erroneous commitment in the event of an L1 reorg.
 */
struct ForkSpec {
    uint256 blockHeight;
    bytes32 blockHash;
}

/**
 * Accepts an L2 output checkpoint and the timestamp of the corresponding L2
 * block. The timestamp must be equal to the current value returned by
 * `nextTimestamp()` in order to be accepted.
 * @param _l2Output The L2 output of the checkpoint block.
 * @param _timestamp The L2 block timestamp that resulted in _l2Output.
 * @param _forkSpecifier A commitment to a specific fork.
 */
function appendL2Output(bytes32 _l2Output, uint256 _timestamp, ForkSpec _forkSpecifier) external


/**
 * Computes the timestamp of the next L2 block that needs to be checkpointed.
 */
function nextTimestamp() public view returns (uint256) {
    return latestBlockTimestamp + submissionInterval;
}

/**
 * Computes the L2 block number given a target L2 block timestamp.
 * @param _timestamp The L2 block timestamp of the target block.
 */
function computeL2BlockNumber(uint256 _timestamp) public view returns (uint256)
```

## Security Considerations

### L1 Reorgs

If the L1 has a reorg after an output has been generated and submitted, the L2 state and correct output may change
leading to a faulty proposal. This is mitigated against in the OutputOracle by checking that the block at
`_forkSpecifier.blockHeight` has the expected hash `_forkSpecifier.blockHash`.
