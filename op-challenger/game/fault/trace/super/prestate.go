package super

import (
	"context"
	"errors"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// SuperNodePrestateRootProvider is the interface for fetching super roots from a supernode
type SuperNodePrestateRootProvider interface {
	SuperRootAtTimestamp(ctx context.Context, timestamp uint64) (eth.SuperRootAtTimestampResponse, error)
}

type SuperRootPrestateProvider struct {
	provider  RootProvider
	timestamp uint64
}

var _ PreimagePrestateProvider = (*SuperRootPrestateProvider)(nil)

func NewSuperRootPrestateProvider(provider RootProvider, prestateTimestamp uint64) *SuperRootPrestateProvider {
	return &SuperRootPrestateProvider{
		provider:  provider,
		timestamp: prestateTimestamp,
	}
}

func (s *SuperRootPrestateProvider) AbsolutePreStateCommitment(ctx context.Context) (common.Hash, error) {
	prestate, err := s.AbsolutePreState(ctx)
	if err != nil {
		return common.Hash{}, err
	}
	return common.Hash(eth.SuperRoot(prestate)), nil
}

func (s *SuperRootPrestateProvider) AbsolutePreState(ctx context.Context) (eth.Super, error) {
	response, err := s.provider.SuperRootAtTimestamp(ctx, hexutil.Uint64(s.timestamp))
	if errors.Is(err, ethereum.NotFound) {
		return nil, ethereum.NotFound
	} else if err != nil {
		return nil, err
	}
	return response.ToSuper()
}

// SuperNodePrestateProvider provides prestate data from a supernode client
type SuperNodePrestateProvider struct {
	provider  SuperNodePrestateRootProvider
	timestamp uint64
}

var _ PreimagePrestateProvider = (*SuperNodePrestateProvider)(nil)

func NewSuperNodePrestateProvider(provider SuperNodePrestateRootProvider, prestateTimestamp uint64) *SuperNodePrestateProvider {
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
	if errors.Is(err, ethereum.NotFound) {
		return nil, ethereum.NotFound
	} else if err != nil {
		return nil, err
	}
	if response.Data == nil {
		return nil, ethereum.NotFound
	}
	return response.Data.Super, nil
}
