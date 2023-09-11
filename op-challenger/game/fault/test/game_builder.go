package test

import (
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum/go-ethereum/common"
)

type GameBuilder struct {
	builder *ClaimBuilder
	Game    types.Game
}

func (c *ClaimBuilder) GameBuilder(agreeWithOutputRoot bool, rootCorrect bool) *GameBuilder {
	return &GameBuilder{
		builder: c,
		Game:    types.NewGameState(agreeWithOutputRoot, c.CreateRootClaim(rootCorrect), uint64(c.maxDepth)),
	}
}

type GameBuilderSeq struct {
	builder   *ClaimBuilder
	lastClaim types.Claim
	game      types.Game
}

func (g *GameBuilder) Seq() *GameBuilderSeq {
	return &GameBuilderSeq{
		builder:   g.builder,
		game:      g.Game,
		lastClaim: g.Game.Claims()[0],
	}
}

func (s *GameBuilderSeq) AttackCorrect() *GameBuilderSeq {
	claim := s.builder.AttackClaim(s.lastClaim, true)
	claim.ContractIndex = len(s.game.Claims())
	s.builder.require.NoError(s.game.Put(claim))
	return &GameBuilderSeq{
		builder:   s.builder,
		game:      s.game,
		lastClaim: claim,
	}
}

func (s *GameBuilderSeq) Attack(value common.Hash) *GameBuilderSeq {
	claim := s.builder.AttackClaimWithValue(s.lastClaim, value)
	claim.ContractIndex = len(s.game.Claims())
	s.builder.require.NoError(s.game.Put(claim))
	return &GameBuilderSeq{
		builder:   s.builder,
		game:      s.game,
		lastClaim: claim,
	}
}

func (s *GameBuilderSeq) DefendCorrect() *GameBuilderSeq {
	claim := s.builder.DefendClaim(s.lastClaim, true)
	claim.ContractIndex = len(s.game.Claims())
	s.builder.require.NoError(s.game.Put(claim))
	return &GameBuilderSeq{
		builder:   s.builder,
		game:      s.game,
		lastClaim: claim,
	}
}

func (s *GameBuilderSeq) Defend(value common.Hash) *GameBuilderSeq {
	claim := s.builder.DefendClaimWithValue(s.lastClaim, value)
	claim.ContractIndex = len(s.game.Claims())
	s.builder.require.NoError(s.game.Put(claim))
	return &GameBuilderSeq{
		builder:   s.builder,
		game:      s.game,
		lastClaim: claim,
	}
}
