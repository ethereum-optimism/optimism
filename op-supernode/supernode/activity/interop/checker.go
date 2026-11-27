package interop

import (
	"context"
	"fmt"
	"slices"
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
	L1BlockRefsByNumbers(ctx context.Context, nums []uint64) ([]eth.L1BlockRef, error)
}

// l1FinalizedSource is optionally implemented by an l1ByNumberSource. When it
// is, the checker caches the canonical hash of finalized L1 blocks.
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
// changes. When the source can report finality, lookups at or below the last
// known finalized number are answered from a cache. During catch-up, every
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

func newL1ConsistencyChecker(l1 l1ByNumberSource) l1ConsistencyChecker {
	if l1 == nil {
		return nil
	}
	c := &l1ByNumberChecker{l1: l1, now: time.Now}
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
// It fetches the heads that are not in the finalized cache in one batch request.
func (c *l1ByNumberChecker) SameL1Chain(ctx context.Context, heads []eth.BlockID) (bool, error) {
	var toFetch []eth.BlockID
	var nums []uint64
	checked := make(map[eth.BlockID]struct{}, len(heads))
	for _, head := range heads {
		if head == (eth.BlockID{}) {
			continue
		}
		if _, ok := checked[head]; ok {
			continue
		}
		checked[head] = struct{}{}
		if hash, ok := c.cachedCanonicalHash(head.Number); ok {
			if hash != head.Hash {
				return false, nil
			}
			continue
		}
		toFetch = append(toFetch, head)
		nums = append(nums, head.Number)
	}
	if len(toFetch) == 0 {
		return true, nil
	}

	// Read finality before fetching: a hash fetched while its number was above
	// finality must not be cached, even if finality passes it meanwhile.
	finalized, canCache := c.finalizedNumber(ctx, slices.Max(nums))
	refs, err := c.l1.L1BlockRefsByNumbers(ctx, nums)
	if err != nil {
		return false, err
	}
	if len(refs) != len(nums) {
		return false, fmt.Errorf("expected %d L1 block refs, got %d", len(nums), len(refs))
	}
	for i, head := range toFetch {
		if canCache && head.Number <= finalized {
			c.canonical.Add(head.Number, refs[i].Hash)
		}
		if refs[i].Hash != head.Hash {
			return false, nil
		}
	}
	return true, nil
}

func (c *l1ByNumberChecker) cachedCanonicalHash(num uint64) (common.Hash, bool) {
	if c.canonical == nil {
		return common.Hash{}, false
	}
	return c.canonical.Get(num)
}

// finalizedNumber returns the highest known L1 finalized number, and false when
// the source cannot report finality. It refreshes the value at most once per
// finalizedRefreshInterval, and only when num is above it. An error keeps the
// known value, which only costs cache misses.
func (c *l1ByNumberChecker) finalizedNumber(ctx context.Context, num uint64) (uint64, bool) {
	if c.finalized == nil {
		return 0, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if num <= c.finalizedNum {
		return c.finalizedNum, true
	}
	if !c.finalizedSeenAt.IsZero() && c.now().Sub(c.finalizedSeenAt) < finalizedRefreshInterval {
		return c.finalizedNum, true
	}
	if ref, err := c.finalized.L1BlockRefByLabel(ctx, eth.Finalized); err == nil {
		c.finalizedSeenAt = c.now()
		c.finalizedNum = max(c.finalizedNum, ref.Number)
	}
	return c.finalizedNum, true
}
