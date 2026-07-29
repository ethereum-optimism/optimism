package testutils

import "github.com/ethereum/go-ethereum/common"

type MockRuntimeConfig struct {
	P2PSeqAddress     common.Address
	PrevP2PSeqAddress common.Address
	Confirmed         bool
}

func (m *MockRuntimeConfig) P2PSequencerAddress() common.Address {
	return m.P2PSeqAddress
}

func (m *MockRuntimeConfig) PreviousP2PSequencerAddress() common.Address {
	return m.PrevP2PSeqAddress
}

func (m *MockRuntimeConfig) ConfirmCurrentSigner() {
	m.Confirmed = true
}
