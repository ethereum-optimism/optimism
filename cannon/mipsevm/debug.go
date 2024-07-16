package mipsevm

type DebugInfo struct {
	Pages               int `json:"pages"`
	NumPreimageRequests int `json:"num_preimage_requests"`
	TotalPreimageSize   int `json:"total_preimage_size"`
}
