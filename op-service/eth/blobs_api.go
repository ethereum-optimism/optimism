package eth

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
	SecondsPerSlot Uint64String `json:"SECONDS_PER_SLOT"`
}

type APIConfigResponse struct {
	Data ReducedConfigData `json:"data"`
}
