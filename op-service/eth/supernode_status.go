package eth

// SuperNodeSyncStatusResponse is the response returned by supernode_syncStatus.
type SuperNodeSyncStatusResponse struct {
	// Chains contains the per-chain op-node sync status.
	Chains map[ChainID]SyncStatus `json:"chains"`

	// ChainIDs are the chain IDs in the dependency set, sorted ascending.
	ChainIDs []ChainID `json:"chain_ids"`

	// CurrentL1 is the L1 block currently being processed by the slowest L1
	// processor (op-node or verifier) in the supernode. Every L1 block strictly
	// below CurrentL1.Number has been fully processed by all chains; data at
	// CurrentL1 itself may still be incomplete. Consumers gating on
	// "L1[≤X] is fully processed" must require CurrentL1.Number > X.
	// Aggregated as the minimum of per-chain current L1 block IDs, including verifiers.
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
