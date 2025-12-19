package eth

// OutputWithRequiredL1 is the full Output and its source L1 block
type OutputWithRequiredL1 struct {
	Output     *OutputResponse `json:"output"`
	RequiredL1 BlockID         `json:"required_l1"`
}

type SuperRootResponseData struct {

	// VerifiedRequiredL1 is the minimum L1 block including the required data to fully verify all blocks at this timestamp
	VerifiedRequiredL1 BlockID `json:"verified_required_l1"`

	// Super is the unhashed data for the superroot at the given timestamp after all verification is applied.
	Super Super `json:"super"`

	// SuperRoot is the superroot at the given timestamp after all verification is applied.
	SuperRoot Bytes32 `json:"super_root"`
}

// AtTimestampResponse is the response superroot_atTimestamp
type SuperRootAtTimestampResponse struct {
	// CurrentL1 is the highest L1 block that has been fully derived and verified by all chains.
	CurrentL1 BlockID `json:"current_l1"`

	// OptimisticAtTimestamp is the L2 block that would be applied if verification were assumed to be successful,
	// and the minimum L1 block required to derive them. If Data is nil, some chains may be absent from this map,
	// indicating that there is no optimistic block for the chain at the requested timestamp that can be derived
	// from the L1 data currently processed.
	OptimisticAtTimestamp map[ChainID]OutputWithRequiredL1 `json:"optimistic_at_timestamp"`

	// ChainIDs are the chain IDs in the dependency set at the requested timestamp, sorted ascending.
	ChainIDs []ChainID `json:"chain_ids"`

	// Data provides information about the super root at the requested timestamp if present. If block data at the
	// requested timestamp is not present, the data will be nil.
	Data *SuperRootResponseData `json:"data,omitempty"`
}
