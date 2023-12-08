package cannon

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm"
	"github.com/ethereum-optimism/optimism/op-challenger/config"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
)

var _ types.PrestateProvider = (*CannonPrestateProvider)(nil)

type CannonPrestateProvider struct {
	prestate string
}

func NewPrestateProvider(cfg *config.Config) *CannonPrestateProvider {
	return &CannonPrestateProvider{
		prestate: cfg.CannonAbsolutePreState,
	}
}

func (p *CannonPrestateProvider) absolutePreState() ([]byte, error) {
	state, err := parseState(p.prestate)
	if err != nil {
		return nil, fmt.Errorf("cannot load absolute pre-state: %w", err)
	}
	return state.EncodeWitness(), nil
}

func (p *CannonPrestateProvider) AbsolutePreStateCommitment(_ context.Context) (common.Hash, error) {
	state, err := p.absolutePreState()
	if err != nil {
		return common.Hash{}, fmt.Errorf("cannot load absolute pre-state: %w", err)
	}
	hash, err := mipsevm.StateWitness(state).StateHash()
	if err != nil {
		return common.Hash{}, fmt.Errorf("cannot hash absolute pre-state: %w", err)
	}
	return hash, nil
}
