// Dafny model for the ChainContainer interface from chain_container/chain_container.go.
// Models the subset of the interface used by the Interop activity.

include "VerifiedDB.dfy"

module ChainContainer {
  import opened Types
  import opened VerifiedDB

  // Result of a single OptimisticAt query.
  // Corresponds to the l2Block and l1Block values returned by c.OptimisticAt in
  // checkChainsReady in interop.go.
  datatype OptimisticAtResult = OptimisticAtResult(l2Block: BlockID, l1Head: BlockID)

  // Combined result of a successful FetchReceipts call.
  // Corresponds to (eth.BlockInfo, types.Receipts) returned by ChainContainer.FetchReceipts
  // in chain_container.go.
  datatype FetchReceiptsResult = FetchReceiptsResult(info: BlockInfo, logs: BlockLogs)

  class ChainContainer {

    // Returns the optimistic L2 block and its L1 inclusion head at the given
    // timestamp, or None if the chain has no block at that timestamp yet
    // (corresponds to an ethereum.NotFound error in Go).
    // Corresponds to ChainContainer.OptimisticAt in chain_container.go.
    method OptimisticAt(ts: nat) returns (result: Option<OptimisticAtResult>)
      ensures {:axiom} result.Some? ==>
        BlockInfo(result.value.l2Block).Some? &&
        BlockInfo(result.value.l2Block).value.timestamp <= ts

    // Prunes deny-list entries at or after the given timestamp.
    // Pure I/O with no modeled state changes.
    // Corresponds to ChainContainer.PruneDeniedAtOrAfterTimestamp in chain_container.go.
    method PruneDeniedAtOrAfterTimestamp(timestamp: nat)
      modifies this

    // Rewinds the chain engine to the given timestamp.
    // Returns false if the operation failed.
    // Pure I/O with no modeled state changes.
    // Corresponds to ChainContainer.RewindEngine in chain_container.go.
    method RewindEngine(resetTo: nat) returns (success: bool)
      modifies this

    // Adds a block to the deny list and potentially triggers a chain rewind.
    // Returns false if the operation failed.
    // Pure I/O with no modeled state changes.
    // Corresponds to ChainContainer.InvalidateBlock in chain_container.go.
    method InvalidateBlock(blockID: BlockID, timestamp: nat) returns (success: bool)
      modifies this

    // Fetches block info and logs for the given block.
    // Returns None if the fetch failed (corresponds to an error return in Go).
    // In the model, receipts are abstracted to executing messages only; see BlockLogs.
    // Corresponds to ChainContainer.FetchReceipts in chain_container.go.
    method FetchReceipts(blockID: BlockID) returns (result: Option<FetchReceiptsResult>)
      // But not <==>. BlockInfo and BlockLogs also include blocks that are no longer
      // part of the canonical chain, so FetchReceipts might return None for those.
      ensures {:axiom} result.Some? ==> BlockInfo(blockID).Some?
      ensures {:axiom} result.Some? ==> BlockLogs(blockID).Some?
      ensures {:axiom} result.Some? ==> result.value.info == BlockInfo(blockID).value
      ensures {:axiom} result.Some? ==> result.value.logs == BlockLogs(blockID).value

    // Returns the block time for this chain in seconds. Immutable per-chain configuration.
    // Corresponds to cc.InteropChain.BlockTime() in chain_container.go.
    function BlockTime(): nat
      reads {}

    // BlockInfo and BlockLogs represent an immutable source of truth for block information,
    // independently of whether the block is part of the current chain or not.
    ghost function BlockInfo(blockID: BlockID) : Option<BlockInfo>
      reads {}
      ensures {:axiom} BlockInfo(blockID).Some? ==> BlockInfo(blockID).value.id == blockID

    ghost function BlockLogs(blockID: BlockID) : Option<BlockLogs>
      reads {}
      ensures {:axiom} BlockLogs(blockID).Some? <==> BlockInfo(blockID).Some?
  }
}
