package solver

import (
	"github.com/ethereum-optimism/optimism/op-challenger/fault/types"
	"github.com/ethereum/go-ethereum/common"
)

// VerboseProvider is a [TraceProvider] that exposes opinionated methods for claims.
type VerboseProvider interface {
	types.TraceProvider
	AgreeWithClaim(claim types.ClaimData, gameDepth int) (bool, error)
	CounterClaim(claim types.Claim, position types.Position, gameDepth int) (*types.Claim, error)
}

type solverProvider struct {
	types.TraceProvider
}

// NewSolverProvider returns a new [VerboseProvider] that uses the given [types.TraceProvider].
func NewSolverProvider(provider types.TraceProvider) VerboseProvider {
	return &solverProvider{provider}
}

// AgreeWithClaim returns true if the claim is valid, false otherwise.
func (s *solverProvider) AgreeWithClaim(claim types.ClaimData, gameDepth int) (bool, error) {
	ourValue, err := s.traceAtPosition(claim.Position, gameDepth)
	return ourValue == claim.Value, err
}

// traceAtPosition returns the trace at the given position.
func (s *solverProvider) traceAtPosition(position types.Position, gameDepth int) (common.Hash, error) {
	index := position.TraceIndex(gameDepth)
	hash, err := s.Get(index)
	return hash, err
}

// CounterClaim returns a counter claim at the given [Position].
func (s *solverProvider) CounterClaim(claim types.Claim, position types.Position, gameDepth int) (*types.Claim, error) {
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
