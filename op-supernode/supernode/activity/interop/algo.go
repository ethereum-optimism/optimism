package interop

import (
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// verifyInteropMessages validates all executing messages at the given timestamp.
// Returns a Result indicating whether all messages are valid or which chains have invalid blocks.
func (i *Interop) verifyInteropMessages(ts uint64, blocksAtTimestamp map[eth.ChainID]eth.BlockID) (Result, error) {
	// TODO(#18743): Implement message verification
	// For now, return all blocks as valid (stub behavior)
	result := Result{Timestamp: ts, L2Heads: make(map[eth.ChainID]eth.BlockID)}
	for _, chain := range i.chains {
		blockID := blocksAtTimestamp[chain.ID()]
		result.L2Heads[chain.ID()] = blockID
	}
	return result, nil
}
