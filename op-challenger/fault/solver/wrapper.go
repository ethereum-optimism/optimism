package solver

import (
	"github.com/ethereum-optimism/optimism/op-challenger/fault/types"
	"github.com/ethereum/go-ethereum/common"
)

// ProviderWrapper is a [TraceProvider] that exposes opinionated methods for claims.
type ProviderWrapper interface {
	types.TraceProvider
	AgreeWithClaim(claim types.ClaimData, gameDepth int) (bool, error)
	Attack(claim types.Claim, gameDepth int) (*types.Claim, error)
	Defend(claim types.Claim, gameDepth int) (*types.Claim, error)
}

type Wrapper struct {
	types.TraceProvider
}

// NewProviderWrapper returns a new [ProviderWrapper] that uses the given [types.TraceProvider].
func NewProviderWrapper(provider types.TraceProvider) ProviderWrapper {
	return &Wrapper{provider}
}

// AgreeWithClaim returns true if the claim is valid, false otherwise.
func (s *Wrapper) AgreeWithClaim(claim types.ClaimData, gameDepth int) (bool, error) {
	ourValue, err := s.traceAtPosition(claim.Position, gameDepth)
	return ourValue == claim.Value, err
}

// traceAtPosition returns the trace at the given position.
func (s *Wrapper) traceAtPosition(position types.Position, gameDepth int) (common.Hash, error) {
	index := position.TraceIndex(gameDepth)
	hash, err := s.Get(index)
	return hash, err
}

// counterClaim returns a counter claim at the given [Position].
func (s *Wrapper) counterClaim(claim types.Claim, position types.Position, gameDepth int) (*types.Claim, error) {
	value, err := s.traceAtPosition(position, gameDepth)
	if err != nil {
		return nil, err
	}
	return &types.Claim{
		ClaimData:           types.ClaimData{Value: value, Position: position},
		Parent:              claim.ClaimData,
		ParentContractIndex: claim.ContractIndex,
	}, nil
}

// Attack returns a new claim that attacks the given claim.
func (s *Wrapper) Attack(claim types.Claim, gameDepth int) (*types.Claim, error) {
	return s.counterClaim(claim, claim.Attack(), gameDepth)
}

// Defend returns a new claim that defends the given claim.
func (s *Wrapper) Defend(claim types.Claim, gameDepth int) (*types.Claim, error) {
	if claim.IsRoot() {
		return nil, nil
	}
	return s.counterClaim(claim, claim.Defend(), gameDepth)
}
