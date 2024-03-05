package superchain

import (
	"github.com/ethereum-optimism/optimism/op-test/components/l1"
)

type StandardSuperchain struct {
	l1 l1.L1
}

func (s *StandardSuperchain) L1() l1.L1 {
	return s.l1
}

var _ Superchain = (*StandardSuperchain)(nil)
