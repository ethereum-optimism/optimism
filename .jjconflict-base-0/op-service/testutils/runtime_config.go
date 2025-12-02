package testutils

import "github.com/ethereum/go-ethereum/common"

type MockRuntimeConfig struct {
	P2PSeqAddress common.Address
}

func (m *MockRuntimeConfig) P2PSequencerAddress() common.Address {
	return m.P2PSeqAddress
}
