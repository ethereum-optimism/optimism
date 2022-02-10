# L2 output root Proposals Specification

<!-- All glossary references in this file. -->
[g-rollup-node]: glossary.md#rollup-node

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**

- [Proposing L2 output commitments](#proposing-l2-output-commitments)
- [L2 output commitment construction](#l2-output-commitment-construction)

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

## Proposing L2 output commitments

The proposer's role is to construct and submit output commitments on a configurable interval to a contract on L1.

TODO: describe integration with rollup node and L2 execution engine (see PR #179).

TODO: link to contract specification/source of L2 Output oracle on L1.

## L2 output commitment construction

This merkle-structure is defined with [SSZ], a type system for merkleization and serialization, used in
L1 (beacon-chain). However, we replace `sha256` with `keccak256` to save gas costs in the EVM.

[SSZ]: https://github.com/ethereum/consensus-specs/blob/dev/ssz/simple-serialize.md

```python
class L2Output(Container):
  state_root: Bytes32
  withdrawal_storage_root: Bytes32  # TODO: withdrawals specifcation work-in-progress
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
