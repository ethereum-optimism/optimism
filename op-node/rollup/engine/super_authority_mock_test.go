package engine

import (
	"fmt"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
)

// mockSuperAuthority implements SuperAuthority for testing.
type mockSuperAuthority struct {
	fullyVerifiedL2Head eth.BlockID
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

var _ rollup.SuperAuthority = (*mockSuperAuthority)(nil)
