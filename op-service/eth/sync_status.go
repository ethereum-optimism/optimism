package eth

// SyncStatus is a snapshot of the driver.
// Values may be zeroed if not yet initialized.
type SyncStatus struct {
	// CurrentL1 is the L1 block that the derivation process is currently at in the inner-most stage.
	// This may not be fully derived into L2 data yet.
	// The safe L2 blocks were produced/included fully from the L1 chain up to and including this L1 block.
	// If the node is synced, this matches the HeadL1, minus the verifier confirmation distance.
	CurrentL1 L1BlockRef `json:"current_l1"`
	// CurrentL1Finalized is the L1 block that the derivation process is currently accepting as finalized
	// in the inner-most stage,
	// This may not be fully derived into L2 data yet.
	// The finalized L2 blocks were produced/included fully from the L1 chain up to and including this L1 block.
	// This may lag behind the FinalizedL1 when the FinalizedL1 could not yet be verified
	// to be canonical w.r.t. the currently derived L2 chain. It may be zeroed if no block could be verified yet.
	CurrentL1Finalized L1BlockRef `json:"current_l1_finalized"`
	// HeadL1 is the perceived head of the L1 chain, no confirmation distance.
	// The head is not guaranteed to build on the other L1 sync status fields,
	// as the node may be in progress of resetting to adapt to a L1 reorg.
	HeadL1      L1BlockRef `json:"head_l1"`
	SafeL1      L1BlockRef `json:"safe_l1"`
	FinalizedL1 L1BlockRef `json:"finalized_l1"`
	// UnsafeL2 is the absolute tip of the L2 chain,
	// pointing to block data that has not been submitted to L1 yet.
	// The sequencer is building this, and verifiers may also be ahead of the
	// SafeL2 block if they sync blocks via p2p or other offchain sources.
	UnsafeL2 L2BlockRef `json:"unsafe_l2"`
	// SafeL2 points to the L2 block that was derived from the L1 chain.
	// This point may still reorg if the L1 chain reorgs.
	SafeL2 L2BlockRef `json:"safe_l2"`
	// FinalizedL2 points to the L2 block that was derived fully from
	// finalized L1 information, thus irreversible.
	FinalizedL2 L2BlockRef `json:"finalized_l2"`
	// PendingSafeL2 points to the L2 block processed from the batch, but not consolidated to the safe block yet.
	PendingSafeL2 L2BlockRef `json:"pending_safe_l2"`
	// UnsafeL2SyncTarget points to the first unprocessed unsafe L2 block.
	// It may be zeroed if there is no targeted block.
	UnsafeL2SyncTarget L2BlockRef `json:"queued_unsafe_l2"`
}
