package test

import (
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum/go-ethereum/common"
)

type GameBuilder struct {
	builder         *ClaimBuilder
	Game            types.Game
	ExpectedActions []types.Action
}

func (c *ClaimBuilder) GameBuilder(agreeWithOutputRoot bool, rootCorrect bool) *GameBuilder {
	return &GameBuilder{
		builder: c,
		Game:    types.NewGameState(agreeWithOutputRoot, c.CreateRootClaim(rootCorrect), uint64(c.maxDepth)),
	}
}

type GameBuilderSeq struct {
	gameBuilder *GameBuilder
	builder     *ClaimBuilder
	lastClaim   types.Claim
	game        types.Game
}

func (g *GameBuilder) Seq() *GameBuilderSeq {
	return &GameBuilderSeq{
		gameBuilder: g,
		builder:     g.builder,
		game:        g.Game,
		lastClaim:   g.Game.Claims()[0],
	}
}

func (s *GameBuilderSeq) AttackCorrect() *GameBuilderSeq {
	claim := s.builder.AttackClaim(s.lastClaim, true)
	claim.ContractIndex = len(s.game.Claims())
	s.builder.require.NoError(s.game.Put(claim))
	return &GameBuilderSeq{
		gameBuilder: s.gameBuilder,
		builder:     s.builder,
		game:        s.game,
		lastClaim:   claim,
	}
}

func (s *GameBuilderSeq) Attack(value common.Hash) *GameBuilderSeq {
	claim := s.builder.AttackClaimWithValue(s.lastClaim, value)
	claim.ContractIndex = len(s.game.Claims())
	s.builder.require.NoError(s.game.Put(claim))
	return &GameBuilderSeq{
		gameBuilder: s.gameBuilder,
		builder:     s.builder,
		game:        s.game,
		lastClaim:   claim,
	}
}

func (s *GameBuilderSeq) DefendCorrect() *GameBuilderSeq {
	claim := s.builder.DefendClaim(s.lastClaim, true)
	claim.ContractIndex = len(s.game.Claims())
	s.builder.require.NoError(s.game.Put(claim))
	return &GameBuilderSeq{
		gameBuilder: s.gameBuilder,
		builder:     s.builder,
		game:        s.game,
		lastClaim:   claim,
	}
}

func (s *GameBuilderSeq) Defend(value common.Hash) *GameBuilderSeq {
	claim := s.builder.DefendClaimWithValue(s.lastClaim, value)
	claim.ContractIndex = len(s.game.Claims())
	s.builder.require.NoError(s.game.Put(claim))
	return &GameBuilderSeq{
		gameBuilder: s.gameBuilder,
		builder:     s.builder,
		game:        s.game,
		lastClaim:   claim,
	}
}

func (s *GameBuilderSeq) ExpectAttack() *GameBuilderSeq {
	newPos := s.lastClaim.Position.Attack()
	value := s.builder.CorrectClaimAtPosition(newPos)
	s.gameBuilder.ExpectedActions = append(s.gameBuilder.ExpectedActions, types.Action{
		Type:      types.ActionTypeMove,
		ParentIdx: s.lastClaim.ContractIndex,
		IsAttack:  true,
		Value:     value,
	})
	return s
}

func (s *GameBuilderSeq) ExpectDefend() *GameBuilderSeq {
	newPos := s.lastClaim.Position.Defend()
	value := s.builder.CorrectClaimAtPosition(newPos)
	s.gameBuilder.ExpectedActions = append(s.gameBuilder.ExpectedActions, types.Action{
		Type:      types.ActionTypeMove,
		ParentIdx: s.lastClaim.ContractIndex,
		IsAttack:  false,
		Value:     value,
	})
	return s
}

func (s *GameBuilderSeq) ExpectStepAttack() *GameBuilderSeq {
	traceIdx := s.lastClaim.TraceIndex(s.builder.maxDepth)
	s.gameBuilder.ExpectedActions = append(s.gameBuilder.ExpectedActions, types.Action{
		Type:       types.ActionTypeStep,
		ParentIdx:  s.lastClaim.ContractIndex,
		IsAttack:   true,
		PreState:   s.builder.CorrectPreState(traceIdx),
		ProofData:  s.builder.CorrectProofData(traceIdx),
		OracleData: s.builder.CorrectOracleData(traceIdx),
	})
	return s
}

func (s *GameBuilderSeq) ExpectStepDefend() *GameBuilderSeq {
	traceIdx := s.lastClaim.TraceIndex(s.builder.maxDepth) + 1
	s.gameBuilder.ExpectedActions = append(s.gameBuilder.ExpectedActions, types.Action{
		Type:       types.ActionTypeStep,
		ParentIdx:  s.lastClaim.ContractIndex,
		IsAttack:   false,
		PreState:   s.builder.CorrectPreState(traceIdx),
		ProofData:  s.builder.CorrectProofData(traceIdx),
		OracleData: s.builder.CorrectOracleData(traceIdx),
	})
	return s
}
