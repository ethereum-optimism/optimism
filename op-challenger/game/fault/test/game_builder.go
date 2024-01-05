package test

import (
	"math/big"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum/go-ethereum/common"
)

type GameBuilder struct {
	builder         *ClaimBuilder
	Game            types.Game
	ExpectedActions []types.Action
}

func (c *ClaimBuilder) GameBuilder(rootCorrect bool) *GameBuilder {
	return &GameBuilder{
		builder: c,
		Game:    types.NewGameState([]types.Claim{c.CreateRootClaim(rootCorrect)}, uint64(c.maxDepth)),
	}
}

type GameBuilderSeq struct {
	gameBuilder *GameBuilder
	builder     *ClaimBuilder
	lastClaim   types.Claim
}

func (g *GameBuilder) Seq() *GameBuilderSeq {
	return g.SeqFrom(g.Game.Claims()[0])
}

func (g *GameBuilder) SeqFrom(claim types.Claim) *GameBuilderSeq {
	return &GameBuilderSeq{
		gameBuilder: g,
		builder:     g.builder,
		lastClaim:   claim,
	}
}

// addClaimToGame replaces the game being built with a new instance that has claim as the latest claim.
// The ContractIndex in claim is updated with its position in the game's claim array.
func (s *GameBuilderSeq) addClaimToGame(claim *types.Claim) {
	claim.ContractIndex = len(s.gameBuilder.Game.Claims())
	claims := append(s.gameBuilder.Game.Claims(), *claim)
	s.gameBuilder.Game = types.NewGameState(claims, uint64(s.builder.maxDepth))
}

func (s *GameBuilderSeq) AttackCorrect() *GameBuilderSeq {
	claim := s.builder.AttackClaim(s.lastClaim, true)
	s.addClaimToGame(&claim)
	return &GameBuilderSeq{
		gameBuilder: s.gameBuilder,
		builder:     s.builder,
		lastClaim:   claim,
	}
}

func (s *GameBuilderSeq) Attack(value common.Hash) *GameBuilderSeq {
	claim := s.builder.AttackClaimWithValue(s.lastClaim, value)
	s.addClaimToGame(&claim)
	return &GameBuilderSeq{
		gameBuilder: s.gameBuilder,
		builder:     s.builder,
		lastClaim:   claim,
	}
}

func (s *GameBuilderSeq) DefendCorrect() *GameBuilderSeq {
	claim := s.builder.DefendClaim(s.lastClaim, true)
	s.addClaimToGame(&claim)
	return &GameBuilderSeq{
		gameBuilder: s.gameBuilder,
		builder:     s.builder,
		lastClaim:   claim,
	}
}

func (s *GameBuilderSeq) Defend(value common.Hash) *GameBuilderSeq {
	claim := s.builder.DefendClaimWithValue(s.lastClaim, value)
	s.addClaimToGame(&claim)
	return &GameBuilderSeq{
		gameBuilder: s.gameBuilder,
		builder:     s.builder,
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
	traceIdx := new(big.Int).Add(s.lastClaim.TraceIndex(s.builder.maxDepth), big.NewInt(1))
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
