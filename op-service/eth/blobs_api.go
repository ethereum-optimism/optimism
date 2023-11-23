package eth

type BlobSidecar struct {
	BlockRoot     Bytes32      `json:"block_root"`
	Slot          Uint64String `json:"slot"`
	Blob          Blob         `json:"blob"`
	Index         Uint64String `json:"index"`
	KZGCommitment Bytes48      `json:"kzg_commitment"`
	KZGProof      Bytes48      `json:"kzg_proof"`
}

type APIGetBlobSidecarsResponse struct {
	Data []*BlobSidecar `json:"data"`
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
