package engine

import (
	"fmt"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
)

// mockSuperAuthority implements rollup.SuperAuthority for testing.
//
// Helper fields fullyVerifiedL2Head / finalizedL2Head and the *Source fields
// carry the head returned by the tri-state contract methods. *Status carries
// the VerifierHeadStatus (defaults to VerifierHeadOk).
type mockSuperAuthority struct {
	fullyVerifiedL2Head       eth.BlockID
	fullyVerifiedL2HeadSource rollup.VerifierHeadSource
	fullyVerifiedStatus       rollup.VerifierHeadStatus

	finalizedL2Head       eth.BlockID
	finalizedL2HeadSource rollup.VerifierHeadSource
	finalizedStatus       rollup.VerifierHeadStatus

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

func (m *mockSuperAuthority) FullyVerifiedL2Head() (rollup.VerifierHead, rollup.VerifierHeadStatus) {
	return rollup.VerifierHead{Block: m.fullyVerifiedL2Head, Source: m.fullyVerifiedL2HeadSource}, m.fullyVerifiedStatus
}

func (m *mockSuperAuthority) FinalizedL2Head() (rollup.VerifierHead, rollup.VerifierHeadStatus) {
	return rollup.VerifierHead{Block: m.finalizedL2Head, Source: m.finalizedL2HeadSource}, m.finalizedStatus
}

var _ rollup.SuperAuthority = (*mockSuperAuthority)(nil)
