package executor

import (
	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/cache"
	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/model"
)

// CacheAdapter wraps cache.Resolver to implement CacheResolver.
type CacheAdapter struct {
	resolver *cache.Resolver
	repoRoot string
}

// NewCacheAdapter creates a CacheResolver backed by a real cache.Resolver.
func NewCacheAdapter(resolver *cache.Resolver, repoRoot string) *CacheAdapter {
	return &CacheAdapter{resolver: resolver, repoRoot: repoRoot}
}

func (a *CacheAdapter) ComputeKey(category string, cat model.JobCategoryConfig) (string, error) {
	return a.resolver.ComputeKey(category, cat)
}

func (a *CacheAdapter) Resolve(category string, cat model.JobCategoryConfig) (*CacheResolution, error) {
	res, err := a.resolver.Resolve(category, cat)
	if err != nil {
		return &CacheResolution{}, err
	}
	return &CacheResolution{
		CacheKey: res.CacheKey,
		Hit:      res.Hit,
	}, nil
}

func (a *CacheAdapter) Restore(category string, cat model.JobCategoryConfig) error {
	return a.resolver.Restore(category, cat)
}

func (a *CacheAdapter) Save(category string, cat model.JobCategoryConfig, key string) error {
	return a.resolver.Save(category, cat, key)
}

func (a *CacheAdapter) Verify(cat model.JobCategoryConfig) error {
	return cache.Verify(a.repoRoot, cat)
}
