package cannon

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
)

var _ types.PrestateProvider = (*CannonPrestateProvider)(nil)

type CannonPrestateProvider struct {
	prestate string

	prestateCommitment common.Hash
}

func NewPrestateProvider(prestate string) *CannonPrestateProvider {
	return &CannonPrestateProvider{prestate: prestate}
}

func (p *CannonPrestateProvider) absolutePreState() ([]byte, common.Hash, error) {
	state, err := parseState(p.prestate)
	if err != nil {
		return nil, common.Hash{}, fmt.Errorf("cannot load absolute pre-state: %w", err)
	}
	witness, hash := state.EncodeWitness()
	return witness, hash, nil
}

func (p *CannonPrestateProvider) AbsolutePreStateCommitment(_ context.Context) (common.Hash, error) {
	if p.prestateCommitment != (common.Hash{}) {
		return p.prestateCommitment, nil
	}
	_, hash, err := p.absolutePreState()
	if err != nil {
		return common.Hash{}, fmt.Errorf("cannot load absolute pre-state: %w", err)
	}
	p.prestateCommitment = hash
	return hash, nil
}
