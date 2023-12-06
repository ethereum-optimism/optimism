package testutils

import (
	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type MockRuntimeConfig struct {
	P2PSeqAddress common.Address
}

func (m *MockRuntimeConfig) P2PSequencerAddress(l2Ref eth.L2BlockRef) common.Address {
	return m.P2PSeqAddress
}
