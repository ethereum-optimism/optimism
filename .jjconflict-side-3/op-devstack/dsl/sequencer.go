package dsl

import (
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/seqtypes"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

type TestSequencer struct {
	commonImpl

	inner stack.TestSequencer
}

func NewTestSequencer(inner stack.TestSequencer) *TestSequencer {
	return &TestSequencer{
		commonImpl: commonFromT(inner.T()),
		inner:      inner,
	}
}

func (s *TestSequencer) String() string {
	return s.inner.ID().String()
}

func (s *TestSequencer) Escape() stack.TestSequencer {
	return s.inner
}

func (s *TestSequencer) SequenceBlock(t devtest.T, chainID eth.ChainID, parent common.Hash) {
	ca := s.Escape().ControlAPI(chainID)

	require.NoError(t, ca.New(t.Ctx(), seqtypes.BuildOpts{Parent: parent}))
	require.NoError(t, ca.Next(t.Ctx()))
}
