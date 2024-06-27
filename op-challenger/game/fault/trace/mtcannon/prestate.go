package mtcannon

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
)

var _ types.PrestateProvider = (*MTCannonPrestateProvider)(nil)

type MTCannonPrestateProvider struct {
	prestate string

	prestateCommitment common.Hash
}

func NewPrestateProvider(prestate string) *MTCannonPrestateProvider {
	return &MTCannonPrestateProvider{prestate: prestate}
}

func (p *MTCannonPrestateProvider) absolutePreState() ([]byte, common.Hash, error) {
	state, err := parseState(p.prestate)
	if err != nil {
		return nil, common.Hash{}, fmt.Errorf("cannot load absolute pre-state: %w", err)
	}
	witness, hash := state.EncodeWitness()
	return witness, hash, nil
}

func (p *MTCannonPrestateProvider) AbsolutePreStateCommitment(_ context.Context) (common.Hash, error) {
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
