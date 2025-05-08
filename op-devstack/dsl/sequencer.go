package dsl

import (
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
)

type Sequencer struct {
	commonImpl

	inner stack.Sequencer
}

func NewSequencer(inner stack.Sequencer) *Sequencer {
	return &Sequencer{
		commonImpl: commonFromT(inner.T()),
		inner:      inner,
	}
}

func (s *Sequencer) String() string {
	return s.inner.ID().String()
}

func (s *Sequencer) Escape() stack.Sequencer {
	return s.inner
}
