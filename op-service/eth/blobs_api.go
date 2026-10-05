package eth

import (
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common/hexutil"
)

type BlobSidecar struct {
	Blob          Blob         `json:"blob"`
	Index         Uint64String `json:"index"`
	KZGCommitment Bytes48      `json:"kzg_commitment"`
	KZGProof      Bytes48      `json:"kzg_proof"`
}

type APIBlobSidecar struct {
	Index             Uint64String            `json:"index"`
	Blob              Blob                    `json:"blob"`
	KZGCommitment     Bytes48                 `json:"kzg_commitment"`
	KZGProof          Bytes48                 `json:"kzg_proof"`
	SignedBlockHeader SignedBeaconBlockHeader `json:"signed_block_header"`
	InclusionProof    []Bytes32               `json:"kzg_commitment_inclusion_proof"`
}

func (sc *APIBlobSidecar) BlobSidecar() *BlobSidecar {
	return &BlobSidecar{
		Blob:          sc.Blob,
		Index:         sc.Index,
		KZGCommitment: sc.KZGCommitment,
		KZGProof:      sc.KZGProof,
	}
}

type SignedBeaconBlockHeader struct {
	Message   BeaconBlockHeader `json:"message"`
	Signature hexutil.Bytes     `json:"signature"`
}

type BeaconBlockHeader struct {
	Slot          Uint64String `json:"slot"`
	ProposerIndex Uint64String `json:"proposer_index"`
	ParentRoot    Bytes32      `json:"parent_root"`
	StateRoot     Bytes32      `json:"state_root"`
	BodyRoot      Bytes32      `json:"body_root"`
}

type APIGetBlobSidecarsResponse struct {
	Data []*APIBlobSidecar `json:"data"`
}

type APIBeaconBlobsResponse struct {
	// There are other fields but we only include the ones we're interested in.

	Data []*Blob `json:"data"`
}

type ReducedGenesisData struct {
	GenesisTime Uint64String `json:"genesis_time"`
}

type APIGenesisResponse struct {
	Data ReducedGenesisData `json:"data"`
}

type ReducedConfigData struct {
	SecondsPerSlot Uint64String `json:"SECONDS_PER_SLOT,omitempty"`
	SlotDurationMs Uint64String `json:"SLOT_DURATION_MS,omitempty"`
}

// SlotDurationSeconds returns the slot duration in seconds from SLOT_DURATION_MS, falling back to
// SECONDS_PER_SLOT. A zero value counts as absent. SLOT_DURATION_MS must be a whole number of
// seconds, because callers map L1 block timestamps, which have second granularity, to slots.
//
// The consensus specs replaced SECONDS_PER_SLOT with SLOT_DURATION_MS; beacon nodes may stop
// serving the old key. Port of ethereum-optimism/optimism#23138.
func (d ReducedConfigData) SlotDurationSeconds() (uint64, error) {
	if ms := uint64(d.SlotDurationMs); ms != 0 {
		if ms%1000 != 0 {
			return 0, fmt.Errorf("beacon spec SLOT_DURATION_MS %d is not a whole number of seconds", ms)
		}
		return ms / 1000, nil
	}
	if d.SecondsPerSlot != 0 {
		return uint64(d.SecondsPerSlot), nil
	}
	return 0, errors.New("beacon spec has neither SLOT_DURATION_MS nor SECONDS_PER_SLOT")
}

type APIConfigResponse struct {
	Data ReducedConfigData `json:"data"`
}

type APIVersionResponse struct {
	Data VersionInformation `json:"data"`
}

type VersionInformation struct {
	Version string `json:"version"`
}
