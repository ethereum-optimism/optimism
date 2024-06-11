package asterisc

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum/go-ethereum/common"
)

var _ types.PrestateProvider = (*AsteriscPreStateProvider)(nil)

type AsteriscPreStateProvider struct {
	prestate string

	prestateCommitment common.Hash
}

func NewPrestateProvider(prestate string) *AsteriscPreStateProvider {
	return &AsteriscPreStateProvider{prestate: prestate}
}

func (p *AsteriscPreStateProvider) absolutePreState() (*VMState, error) {
	state, err := parseState(p.prestate)
	if err != nil {
		return nil, fmt.Errorf("cannot load absolute pre-state: %w", err)
	}
	return state, nil
}

func (p *AsteriscPreStateProvider) AbsolutePreStateCommitment(_ context.Context) (common.Hash, error) {
	if p.prestateCommitment != (common.Hash{}) {
		return p.prestateCommitment, nil
	}
	state, err := p.absolutePreState()
	if err != nil {
		return common.Hash{}, fmt.Errorf("cannot load absolute pre-state: %w", err)
	}
	p.prestateCommitment = state.StateHash
	return state.StateHash, nil
}
