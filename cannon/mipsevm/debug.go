package mipsevm

import "github.com/ethereum/go-ethereum/common/hexutil"

type DebugInfo struct {
	Pages               int            `json:"pages"`
	MemoryUsed          hexutil.Uint64 `json:"memory_used"`
	NumPreimageRequests int            `json:"num_preimage_requests"`
	TotalPreimageSize   int            `json:"total_preimage_size"`
}
