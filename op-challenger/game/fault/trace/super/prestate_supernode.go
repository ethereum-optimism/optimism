package super

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
)

type SuperNodePrestateProvider struct {
	provider  SuperNodeRootProvider
	timestamp uint64
}

var _ PreimagePrestateProvider = (*SuperNodePrestateProvider)(nil)

func NewSuperNodePrestateProvider(provider SuperNodeRootProvider, prestateTimestamp uint64) *SuperNodePrestateProvider {
	return &SuperNodePrestateProvider{
		provider:  provider,
		timestamp: prestateTimestamp,
	}
}

func (s *SuperNodePrestateProvider) AbsolutePreStateCommitment(ctx context.Context) (common.Hash, error) {
	prestate, err := s.AbsolutePreState(ctx)
	if err != nil {
		return common.Hash{}, err
	}
	return common.Hash(eth.SuperRoot(prestate)), nil
}

func (s *SuperNodePrestateProvider) AbsolutePreState(ctx context.Context) (eth.Super, error) {
	response, err := s.provider.SuperRootAtTimestamp(ctx, s.timestamp)
	if err != nil {
		return nil, err
	}
	if response.Data == nil {
		return nil, ethereum.NotFound
	}
	return response.Data.Super, nil
}
