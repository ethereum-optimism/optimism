package vm

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
)

var _ types.PrestateProvider = (*PrestateProvider)(nil)

type PrestateProvider struct {
	prestate       string
	stateConverter StateConverter

	prestateCommitment common.Hash
}

func NewPrestateProvider(prestate string, converter StateConverter) *PrestateProvider {
	return &PrestateProvider{
		prestate:       prestate,
		stateConverter: converter,
	}
}

func (p *PrestateProvider) AbsolutePreStateCommitment(ctx context.Context) (common.Hash, error) {
	if p.prestateCommitment != (common.Hash{}) {
		return p.prestateCommitment, nil
	}
	proof, _, _, err := p.stateConverter.ConvertStateToProof(ctx, p.prestate)
	if err != nil {
		return common.Hash{}, fmt.Errorf("cannot load absolute pre-state: %w", err)
	}
	p.prestateCommitment = proof.ClaimValue
	return proof.ClaimValue, nil
}

func (p *PrestateProvider) PrestatePath() string {
	return p.prestate
}
