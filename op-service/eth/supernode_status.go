package eth

// SuperNodeSyncStatusResponse is the response returned by supernode_syncStatus.
type SuperNodeSyncStatusResponse struct {
	// Chains contains the per-chain op-node sync status.
	Chains map[ChainID]SyncStatus `json:"chains"`

	// ChainIDs are the chain IDs in the dependency set, sorted ascending.
	ChainIDs []ChainID `json:"chain_ids"`

	// CurrentL1 is the highest L1 block ID that has been fully derived and verified by all chains.
	// This value is derived from the minimum per-chain current L1 block IDs, including validators.
	CurrentL1 BlockID `json:"current_l1"`

	// SafeTimestamp is the highest L2 timestamp that is safe across the dependency set at the CurrentL1.
	// This value is derived from the minimum per-chain safe L2 head timestamp.
	SafeTimestamp uint64 `json:"safe_timestamp"`

	// LocalSafeTimestamp is the highest L2 timestamp that is local-safe across the dependency set at the CurrentL1.
	// This value is derived from the minimum per-chain local safe L2 head timestamp.
	LocalSafeTimestamp uint64 `json:"local_safe_timestamp"`

	// FinalizedTimestamp is the highest L2 timestamp that is finalized across the dependency set at the CurrentL1.
	// This value is derived from the minimum per-chain finalized L2 head timestamp.
	FinalizedTimestamp uint64 `json:"finalized_timestamp"`
}
