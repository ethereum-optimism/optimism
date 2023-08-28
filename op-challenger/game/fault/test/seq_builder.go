package test

import (
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
)

type SequenceBuilder struct {
	builder   *ClaimBuilder
	lastClaim types.Claim
}

// Seq starts building a claim by following a sequence of attack and defend moves from the root
// The returned SequenceBuilder can be used to add additional moves. e.g:
// claim := Seq(true).Attack(false).Attack(true).Defend(true).Get()
func (c *ClaimBuilder) Seq(rootCorrect bool) *SequenceBuilder {
	claim := c.CreateRootClaim(rootCorrect)
	return &SequenceBuilder{
		builder:   c,
		lastClaim: claim,
	}
}

func (s *SequenceBuilder) Attack(correct bool) *SequenceBuilder {
	claim := s.builder.AttackClaim(s.lastClaim, correct)
	return &SequenceBuilder{
		builder:   s.builder,
		lastClaim: claim,
	}
}

func (s *SequenceBuilder) Defend(correct bool) *SequenceBuilder {
	claim := s.builder.DefendClaim(s.lastClaim, correct)
	return &SequenceBuilder{
		builder:   s.builder,
		lastClaim: claim,
	}
}

func (s *SequenceBuilder) Get() types.Claim {
	return s.lastClaim
}
