package engine

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
)

// mockSuperAuthority implements SuperAuthority for testing.
type mockSuperAuthority struct {
	fullyVerifiedL2Head eth.BlockID
	finalizedL2Head     eth.BlockID
	deniedBlocks        map[uint64]common.Hash
	shouldError         bool
}

func newMockSuperAuthority() *mockSuperAuthority {
	return &mockSuperAuthority{
		deniedBlocks: make(map[uint64]common.Hash),
	}
}

func (m *mockSuperAuthority) denyBlock(blockNumber uint64, hash common.Hash) {
	m.deniedBlocks[blockNumber] = hash
}

func (m *mockSuperAuthority) IsDenied(blockNumber uint64, payloadHash common.Hash) (bool, error) {
	if m.shouldError {
		return false, fmt.Errorf("superauthority check failed")
	}
	deniedHash, exists := m.deniedBlocks[blockNumber]
	if exists && deniedHash == payloadHash {
		return true, nil
	}
	return false, nil
}

func (m *mockSuperAuthority) FullyVerifiedL2Head() (eth.BlockID, bool) {
	return m.fullyVerifiedL2Head, false
}

func (m *mockSuperAuthority) FinalizedL2Head() (eth.BlockID, bool) {
	return m.finalizedL2Head, false
}

// CanonicalDeniedHeight walks the in-memory denied-blocks map from highest to
// lowest block number and returns the lowest height whose denied hash matches
// the canonical hash reported by `canonical`. Mirrors the contract of
// DenyList.CanonicalDeniedHeight for testing.
func (m *mockSuperAuthority) CanonicalDeniedHeight(ctx context.Context, canonical rollup.CanonicalChain) (uint64, bool, error) {
	if m.shouldError {
		return 0, false, fmt.Errorf("superauthority check failed")
	}
	if len(m.deniedBlocks) == 0 {
		return 0, false, nil
	}
	heights := make([]uint64, 0, len(m.deniedBlocks))
	for h := range m.deniedBlocks {
		heights = append(heights, h)
	}
	// Sort descending.
	for i := 1; i < len(heights); i++ {
		for j := i; j > 0 && heights[j] > heights[j-1]; j-- {
			heights[j], heights[j-1] = heights[j-1], heights[j]
		}
	}
	var lowestCanonical uint64
	var found bool
	for _, h := range heights {
		ref, err := canonical.L2BlockRefByNumber(ctx, h)
		if err != nil {
			return lowestCanonical, found, nil
		}
		if ref.Hash != m.deniedBlocks[h] {
			return lowestCanonical, found, nil
		}
		lowestCanonical = h
		found = true
	}
	return lowestCanonical, found, nil
}

var _ rollup.SuperAuthority = (*mockSuperAuthority)(nil)
