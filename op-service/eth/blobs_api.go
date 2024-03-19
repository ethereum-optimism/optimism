package eth

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
	// The inclusion-proof of the blob-sidecar into the beacon-block is ignored,
	// since we verify blobs by their versioned hashes against the execution-layer block instead.
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
	Message BeaconBlockHeader `json:"message"`
	// signature is ignored, since we verify blobs against EL versioned-hashes
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

type ReducedGenesisData struct {
	GenesisTime Uint64String `json:"genesis_time"`
}

type APIGenesisResponse struct {
	Data ReducedGenesisData `json:"data"`
}

type ReducedConfigData struct {
	SecondsPerSlot Uint64String `json:"SECONDS_PER_SLOT"`
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
