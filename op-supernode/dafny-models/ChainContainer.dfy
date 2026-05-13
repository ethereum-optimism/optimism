// Dafny model for the ChainContainer interface from chain_container/chain_container.go.
// Models the subset of the interface used by the Interop activity.

include "VerifiedDB.dfy"

module ChainContainer {
  import opened Types
  import opened VerifiedDB

  class ChainContainer {

    // Returns the optimistic L2 block and its L1 inclusion head at the given
    // timestamp, or None if the chain has no block at that timestamp yet
    // (corresponds to an ethereum.NotFound error in Go).
    // Corresponds to ChainContainer.OptimisticAt in chain_container.go.
    method OptimisticAt(ts: nat) returns (result: Option<OptimisticAtResult>)

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

    // Fetches block info for the given block.
    // In the model, receipts are abstracted away; only the metadata needed by
    // persistFrontierLogs is returned.
    // Corresponds to ChainContainer.FetchReceipts in chain_container.go.
    method FetchReceipts(blockID: BlockID) returns (info: BlockInfo)
      ensures {:axiom} info.id == blockID

  }
}
