package dsl

import (
	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/stack"
)

type Sequencer struct {
	commonImpl

	stack.Sequencer
}

func NewSequencer(inner stack.Sequencer) *Sequencer {
	return &Sequencer{
		commonImpl: commonFromT(inner.T()),
		Sequencer:  inner,
	}
}

func (s *Sequencer) String() string {
	return s.Sequencer.ID().String()
}
