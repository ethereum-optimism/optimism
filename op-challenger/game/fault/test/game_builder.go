package test

import (
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum/go-ethereum/common"
)

type GameBuilder struct {
	builder         *ClaimBuilder
	Game            types.Game
	ExpectedActions []types.Action
}

func NewGameBuilderFromGame(t *testing.T, provider types.TraceProvider, game types.Game) *GameBuilder {
	claimBuilder := NewClaimBuilder(t, game.MaxDepth(), provider)
	return &GameBuilder{
		builder: claimBuilder,
		Game:    types.NewGameState(game.Claims(), game.MaxDepth()),
	}
}

func (c *ClaimBuilder) GameBuilder(rootOpts ...ClaimOpt) *GameBuilder {
	return &GameBuilder{
		builder: c,
		Game:    types.NewGameState([]types.Claim{c.CreateRootClaim(rootOpts...)}, c.maxDepth),
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

func (g *GameBuilderSeq) IsMaxDepth() bool {
	return g.lastClaim.Depth() == g.gameBuilder.Game.MaxDepth()
}

func (g *GameBuilderSeq) IsRoot() bool {
	return g.lastClaim.IsRoot()
}

// addClaimToGame replaces the game being built with a new instance that has claim as the latest claim.
// The ContractIndex in claim is updated with its position in the game's claim array.
// Does nothing if the claim already exists
func (s *GameBuilderSeq) addClaimToGame(claim *types.Claim) {
	if s.gameBuilder.Game.IsDuplicate(*claim) {
		return
	}
	claim.ContractIndex = len(s.gameBuilder.Game.Claims())
	claims := append(s.gameBuilder.Game.Claims(), *claim)
	s.gameBuilder.Game = types.NewGameState(claims, s.builder.maxDepth)
}

func (s *GameBuilderSeq) Attack(opts ...ClaimOpt) *GameBuilderSeq {
	claim := s.builder.AttackClaim(s.lastClaim, opts...)
	s.addClaimToGame(&claim)
	return &GameBuilderSeq{
		gameBuilder: s.gameBuilder,
		builder:     s.builder,
		lastClaim:   claim,
	}
}

func (s *GameBuilderSeq) Defend(opts ...ClaimOpt) *GameBuilderSeq {
	claim := s.builder.DefendClaim(s.lastClaim, opts...)
	s.addClaimToGame(&claim)
	return &GameBuilderSeq{
		gameBuilder: s.gameBuilder,
		builder:     s.builder,
		lastClaim:   claim,
	}
}

func (s *GameBuilderSeq) Step(opts ...ClaimOpt) {
	cfg := newClaimCfg(opts...)
	claimant := DefaultClaimant
	if cfg.claimant != (common.Address{}) {
		claimant = cfg.claimant
	}
	claims := s.gameBuilder.Game.Claims()
	claims[len(claims)-1].CounteredBy = claimant
	s.gameBuilder.Game = types.NewGameState(claims, s.builder.maxDepth)
}

func (s *GameBuilderSeq) ExpectAttack() *GameBuilderSeq {
	newPos := s.lastClaim.Position.Attack()
	value := s.builder.CorrectClaimAtPosition(newPos)
	s.gameBuilder.ExpectedActions = append(s.gameBuilder.ExpectedActions, types.Action{
		Type:        types.ActionTypeMove,
		ParentClaim: s.lastClaim,
		IsAttack:    true,
		Value:       value,
	})
	return s
}

func (s *GameBuilderSeq) ExpectDefend() *GameBuilderSeq {
	newPos := s.lastClaim.Position.Defend()
	value := s.builder.CorrectClaimAtPosition(newPos)
	s.gameBuilder.ExpectedActions = append(s.gameBuilder.ExpectedActions, types.Action{
		Type:        types.ActionTypeMove,
		ParentClaim: s.lastClaim,
		IsAttack:    false,
		Value:       value,
	})
	return s
}

func (s *GameBuilderSeq) ExpectStepAttack() *GameBuilderSeq {
	traceIdx := s.lastClaim.TraceIndex(s.builder.maxDepth)
	s.gameBuilder.ExpectedActions = append(s.gameBuilder.ExpectedActions, types.Action{
		Type:        types.ActionTypeStep,
		ParentClaim: s.lastClaim,
		IsAttack:    true,
		PreState:    s.builder.CorrectPreState(traceIdx),
		ProofData:   s.builder.CorrectProofData(traceIdx),
		OracleData:  s.builder.CorrectOracleData(traceIdx),
	})
	return s
}

func (s *GameBuilderSeq) ExpectStepDefend() *GameBuilderSeq {
	traceIdx := new(big.Int).Add(s.lastClaim.TraceIndex(s.builder.maxDepth), big.NewInt(1))
	s.gameBuilder.ExpectedActions = append(s.gameBuilder.ExpectedActions, types.Action{
		Type:        types.ActionTypeStep,
		ParentClaim: s.lastClaim,
		IsAttack:    false,
		PreState:    s.builder.CorrectPreState(traceIdx),
		ProofData:   s.builder.CorrectProofData(traceIdx),
		OracleData:  s.builder.CorrectOracleData(traceIdx),
	})
	return s
}
