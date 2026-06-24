package engine

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
)

// mockSuperAuthority implements rollup.SuperAuthority for testing.
// holdPrevious{Verified,Finalized} default to false so struct-literal tests
// without explicit setup get the happy-path (ok=true).
type mockSuperAuthority struct {
	fullyVerifiedL2Head       eth.BlockID
	fullyVerifiedTimestamp    uint64
	fullyVerifiedL2HeadSource rollup.VerifierHeadSource
	holdPreviousVerified      bool

	finalizedL2Head       eth.BlockID
	finalizedTimestamp    uint64
	finalizedL2HeadSource rollup.VerifierHeadSource
	holdPreviousFinalized bool

	deniedBlocks map[uint64]common.Hash
	shouldError  bool
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

func (m *mockSuperAuthority) FullyVerifiedL2Head(ctx context.Context) (rollup.VerifierHead, bool) {
	return rollup.VerifierHead{
		Block:     m.fullyVerifiedL2Head,
		Timestamp: m.fullyVerifiedTimestamp,
		Source:    m.fullyVerifiedL2HeadSource,
	}, !m.holdPreviousVerified
}

func (m *mockSuperAuthority) FinalizedL2Head(ctx context.Context) (rollup.VerifierHead, bool) {
	return rollup.VerifierHead{
		Block:     m.finalizedL2Head,
		Timestamp: m.finalizedTimestamp,
		Source:    m.finalizedL2HeadSource,
	}, !m.holdPreviousFinalized
}

var _ rollup.SuperAuthority = (*mockSuperAuthority)(nil)
