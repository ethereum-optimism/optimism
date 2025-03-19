package compare

import (
	"fmt"

	"github.com/ethereum-optimism/optimism/op-fetcher/pkg/fetcher/fetch/script"
)

// CompareProofs compares all proof status fields in chain config with corresponding fields in FetchOutput
func (c *Comparator) CompareProofs() (map[uint64]script.FaultProofStatus, error) {
	result := make(map[uint64]script.FaultProofStatus)

	for chainID, actual := range c.FetchOutput {
		c.lgr.Info("comparing chain info", "chainID", chainID)
		expected, exists := c.ChainList[chainID]
		if !exists {
			return result, fmt.Errorf("chainID %d exists in FetchOutput but not in ChainList", chainID)
		}

		if expected.FaultProofStatus != actual.FaultProofStatus {
			result[chainID] = actual.FaultProofStatus
		}
	}

	return result, nil
}
