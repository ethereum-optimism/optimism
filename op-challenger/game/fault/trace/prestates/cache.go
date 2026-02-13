package prestates

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum-optimism/optimism/op-service/sources/caching"
	"github.com/ethereum/go-ethereum/common"
)

type PrestateSource interface {
	// PrestatePath returns the path to the prestate file to use for the game.
	// The provided prestateHash may be used to differentiate between different states but no guarantee is made that
	// the returned prestate matches the supplied hash.
	PrestatePath(ctx context.Context, prestateHash common.Hash) (string, error)
}

type PrestateProviderCache struct {
	createProvider func(ctx context.Context, prestateHash common.Hash) (types.PrestateProvider, error)
	cache          *caching.LRUCache[common.Hash, types.PrestateProvider]
}

func NewPrestateProviderCache(m caching.Metrics, label string, createProvider func(ctx context.Context, prestateHash common.Hash) (types.PrestateProvider, error)) *PrestateProviderCache {
	return &PrestateProviderCache{
		createProvider: createProvider,
		cache:          caching.NewLRUCache[common.Hash, types.PrestateProvider](m, label, 5),
	}
}

func (p *PrestateProviderCache) GetOrCreate(ctx context.Context, prestateHash common.Hash) (types.PrestateProvider, error) {
	provider, ok := p.cache.Get(prestateHash)
	if ok {
		return provider, nil
	}
	provider, err := p.createProvider(ctx, prestateHash)
	if err != nil {
		return nil, err
	}
	p.cache.Add(prestateHash, provider)
	return provider, nil
}
