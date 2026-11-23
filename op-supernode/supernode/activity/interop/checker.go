package interop

import (
	"context"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	lru "github.com/hashicorp/golang-lru/v2"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

const (
	// finalizedCanonicalCacheSize bounds the number -> canonical hash entries kept
	// for finalized L1 blocks. Catch-up checks roughly one new L1 number every
	// L1 block time of verified L2 time, so this covers well over a day of it.
	finalizedCanonicalCacheSize = 16_384
	// finalizedRefreshInterval rate-limits L1 finality lookups. Finality only
	// moves once per epoch, so a stale value just means a few extra RPCs.
	finalizedRefreshInterval = 12 * time.Second
)

// l1ByNumberSource provides L1 block lookups by number for consistency checking.
type l1ByNumberSource interface {
	L1BlockRefByNumber(ctx context.Context, num uint64) (eth.L1BlockRef, error)
}

// l1FinalizedSource is optionally implemented by an l1ByNumberSource. When it
// is and caching is requested, the checker caches the canonical hash of
// finalized L1 blocks.
type l1FinalizedSource interface {
	L1BlockRefByLabel(ctx context.Context, label eth.BlockLabel) (eth.L1BlockRef, error)
}

// l1ConsistencyChecker verifies that a set of L1 block IDs all belong to
// the same L1 fork.
type l1ConsistencyChecker interface {
	SameL1Chain(ctx context.Context, heads []eth.BlockID) (bool, error)
}

// l1ByNumberChecker is the production l1ConsistencyChecker: it checks each
// head against the canonical L1 chain fetched from an l1ByNumberSource.
//
// A finalized L1 block cannot be reorged, so its number -> hash mapping never
// changes. When caching is enabled (interop.l1-finalized-cache) and the source
// can report finality, lookups at or below the last known finalized number are
// answered from a cache. During catch-up, every
// head is finalized and repeats for many rounds, so this removes most of the
// per-round L1 RPCs. Heads above finality are always fetched, which keeps
// reorg detection at the tip unchanged.
type l1ByNumberChecker struct {
	l1        l1ByNumberSource
	finalized l1FinalizedSource // nil disables the cache

	mu              sync.Mutex
	finalizedNum    uint64
	finalizedSeenAt time.Time
	canonical       *lru.Cache[uint64, common.Hash]
	now             func() time.Time
}

// newL1ConsistencyChecker returns a checker backed by l1. With cacheFinalized,
// it caches canonical hashes of finalized L1 blocks when l1 can report finality.
func newL1ConsistencyChecker(l1 l1ByNumberSource, cacheFinalized bool) l1ConsistencyChecker {
	if l1 == nil {
		return nil
	}
	c := &l1ByNumberChecker{l1: l1, now: time.Now}
	if !cacheFinalized {
		return c
	}
	if f, ok := l1.(l1FinalizedSource); ok {
		cache, err := lru.New[uint64, common.Hash](finalizedCanonicalCacheSize)
		if err == nil {
			c.finalized = f
			c.canonical = cache
		}
	}
	return c
}

// SameL1Chain returns true if all non-zero heads belong to the same canonical L1 chain.
func (c *l1ByNumberChecker) SameL1Chain(ctx context.Context, heads []eth.BlockID) (bool, error) {
	checked := make(map[eth.BlockID]struct{}, len(heads))
	for _, head := range heads {
		if head == (eth.BlockID{}) {
			continue
		}
		if _, ok := checked[head]; ok {
			continue
		}
		checked[head] = struct{}{}
		canonical, err := c.canonicalHash(ctx, head.Number)
		if err != nil {
			return false, err
		}
		if canonical != head.Hash {
			return false, nil
		}
	}
	return true, nil
}

// canonicalHash returns the canonical L1 hash at num, from the cache when num
// is known to be finalized.
func (c *l1ByNumberChecker) canonicalHash(ctx context.Context, num uint64) (common.Hash, error) {
	if c.canonical != nil {
		if hash, ok := c.canonical.Get(num); ok {
			return hash, nil
		}
	}
	// Check finality before fetching: a hash fetched while num was above
	// finality must not be cached, even if finality passes num meanwhile.
	final := c.canonical != nil && c.isFinalized(ctx, num)
	ref, err := c.l1.L1BlockRefByNumber(ctx, num)
	if err != nil {
		return common.Hash{}, err
	}
	if final {
		c.canonical.Add(num, ref.Hash)
	}
	return ref.Hash, nil
}

// isFinalized reports whether num is at or below the L1 finalized head. It
// refreshes the finalized head at most once per finalizedRefreshInterval, and
// only when num is above the last known value. Errors are treated as "not
// finalized", which only costs a cache miss.
func (c *l1ByNumberChecker) isFinalized(ctx context.Context, num uint64) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if num <= c.finalizedNum {
		return true
	}
	if !c.finalizedSeenAt.IsZero() && c.now().Sub(c.finalizedSeenAt) < finalizedRefreshInterval {
		return false
	}
	ref, err := c.finalized.L1BlockRefByLabel(ctx, eth.Finalized)
	if err != nil {
		return false
	}
	c.finalizedSeenAt = c.now()
	if ref.Number > c.finalizedNum {
		c.finalizedNum = ref.Number
	}
	return num <= c.finalizedNum
}
