package eth

import "encoding/json"

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

// superRootResponseDataMarshalling is the JSON marshalling helper for SuperRootResponseData
type superRootResponseDataMarshalling struct {
	VerifiedRequiredL1 BlockID         `json:"verified_required_l1"`
	Super              json.RawMessage `json:"super"`
	SuperRoot          Bytes32         `json:"super_root"`
}

// UnmarshalJSON implements custom JSON unmarshaling for SuperRootResponseData
func (d *SuperRootResponseData) UnmarshalJSON(input []byte) error {
	var dec superRootResponseDataMarshalling
	if err := json.Unmarshal(input, &dec); err != nil {
		return err
	}
	d.VerifiedRequiredL1 = dec.VerifiedRequiredL1
	d.SuperRoot = dec.SuperRoot

	// Unmarshal the Super field - currently only SuperV1 is supported
	if len(dec.Super) > 0 {
		var superV1 SuperV1
		if err := json.Unmarshal(dec.Super, &superV1); err != nil {
			return err
		}
		d.Super = &superV1
	} else {
		d.Super = nil
	}
	return nil
}

// AtTimestampResponse is the response superroot_atTimestamp
type SuperRootAtTimestampResponse struct {
	// CurrentL1 is the highest L1 block that has been fully derived and verified by all chains.
	CurrentL1 BlockID `json:"current_l1"`

	// CurrentSafeTimestamp is the highest L2 timestamp that is safe across the dependency set at the CurrentL1.
	// This value is derived from the minimum per-chain safe L2 head timestamp.
	CurrentSafeTimestamp uint64 `json:"safe_timestamp"`

	// CurrentSafeTimestamp is the highest L2 timestamp that is safe across the dependency set at the CurrentL1.
	// This value is derived from the minimum per-chain local-safe L2 head timestamp.
	CurrentLocalSafeTimestamp uint64 `json:"local_safe_timestamp"`

	// CurrentFinalizedTimestamp is the highest L2 timestamp that is finalized across the dependency set at the CurrentL1.
	// This value is derived from the minimum per-chain finalized L2 head timestamp.
	CurrentFinalizedTimestamp uint64 `json:"finalized_timestamp"`

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
