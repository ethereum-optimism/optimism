package interop

import (
	"context"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

type mockL1Source struct {
	blocks map[uint64]eth.L1BlockRef
}

func (m *mockL1Source) L1BlockRefByNumber(ctx context.Context, num uint64) (eth.L1BlockRef, error) {
	ref, ok := m.blocks[num]
	if !ok {
		return eth.L1BlockRef{}, fmt.Errorf("block %d not found", num)
	}
	return ref, nil
}

// noopL1Checker treats every set of heads as canonical. Intended for tests that
// do not exercise L1 consistency — production must always use the real checker.
type noopL1Checker struct{}

func (noopL1Checker) SameL1Chain(context.Context, []eth.BlockID) (bool, error) {
	return true, nil
}

// inconsistentL1Checker always reports that heads disagree on the canonical L1
// chain. Tests use it to drive observeRound into an L1 inconsistency decision.
type inconsistentL1Checker struct{}

func (inconsistentL1Checker) SameL1Chain(context.Context, []eth.BlockID) (bool, error) {
	return false, nil
}

// staleFrontierL1Checker treats the accepted L1 head as canonical on the first
// check, then reports the frontier L1 heads as stale on the second check.
type staleFrontierL1Checker struct {
	calls int
}

func (s *staleFrontierL1Checker) SameL1Chain(context.Context, []eth.BlockID) (bool, error) {
	s.calls++
	return s.calls == 1, nil
}

func TestL1ConsistencyChecker_SameL1Chain(t *testing.T) {
	t.Parallel()

	hashA := common.HexToHash("0xaaaa")
	hashB := common.HexToHash("0xbbbb")

	source := &mockL1Source{
		blocks: map[uint64]eth.L1BlockRef{
			100: {Hash: hashA, Number: 100},
			200: {Hash: hashB, Number: 200},
		},
	}
	checker := newL1ConsistencyChecker(source)

	t.Run("all match canonical", func(t *testing.T) {
		same, err := checker.SameL1Chain(context.Background(), []eth.BlockID{
			{Hash: hashA, Number: 100},
			{Hash: hashB, Number: 200},
		})
		require.NoError(t, err)
		require.True(t, same)
	})

	t.Run("mismatch detected", func(t *testing.T) {
		same, err := checker.SameL1Chain(context.Background(), []eth.BlockID{
			{Hash: hashA, Number: 100},
			{Hash: common.HexToHash("0xdead"), Number: 200}, // wrong hash
		})
		require.NoError(t, err)
		require.False(t, same)
	})

	t.Run("zero block IDs skipped", func(t *testing.T) {
		same, err := checker.SameL1Chain(context.Background(), []eth.BlockID{
			{},
			{Hash: hashA, Number: 100},
			{},
		})
		require.NoError(t, err)
		require.True(t, same)
	})

	t.Run("empty heads list", func(t *testing.T) {
		same, err := checker.SameL1Chain(context.Background(), []eth.BlockID{})
		require.NoError(t, err)
		require.True(t, same)
	})

	t.Run("nil checker returns nil", func(t *testing.T) {
		nilChecker := newL1ConsistencyChecker(nil)
		require.Nil(t, nilChecker)
	})

	t.Run("source without finality is uncached", func(t *testing.T) {
		c := newL1ConsistencyChecker(source).(*l1ByNumberChecker)
		require.Nil(t, c.canonical)
	})
}

// countingFinalizedL1Source counts by-number and finality lookups.
type countingFinalizedL1Source struct {
	mockL1Source
	finalized      uint64
	byNumberCalls  int
	finalizedCalls int
	onByNumber     func() // runs inside each by-number lookup
}

func (m *countingFinalizedL1Source) L1BlockRefByNumber(ctx context.Context, num uint64) (eth.L1BlockRef, error) {
	m.byNumberCalls++
	if m.onByNumber != nil {
		m.onByNumber()
	}
	return m.mockL1Source.L1BlockRefByNumber(ctx, num)
}

func (m *countingFinalizedL1Source) L1BlockRefByLabel(_ context.Context, label eth.BlockLabel) (eth.L1BlockRef, error) {
	if label != eth.Finalized {
		return eth.L1BlockRef{}, fmt.Errorf("unexpected label %s", label)
	}
	m.finalizedCalls++
	return eth.L1BlockRef{Number: m.finalized}, nil
}

func TestL1ConsistencyChecker_FinalizedCache(t *testing.T) {
	t.Parallel()

	hash := func(n uint64) common.Hash { return common.BigToHash(new(big.Int).SetUint64(n)) }
	newSource := func(finalized uint64) *countingFinalizedL1Source {
		blocks := make(map[uint64]eth.L1BlockRef)
		for n := uint64(90); n <= 110; n++ {
			blocks[n] = eth.L1BlockRef{Hash: hash(n), Number: n}
		}
		return &countingFinalizedL1Source{mockL1Source: mockL1Source{blocks: blocks}, finalized: finalized}
	}
	newChecker := func(src *countingFinalizedL1Source, now *time.Time) *l1ByNumberChecker {
		c := newL1ConsistencyChecker(src).(*l1ByNumberChecker)
		require.NotNil(t, c.canonical)
		c.now = func() time.Time { return *now }
		return c
	}
	ctx := context.Background()

	t.Run("finalized numbers are served from cache", func(t *testing.T) {
		now := time.Unix(1000, 0)
		src := newSource(100)
		c := newChecker(src, &now)
		for i := 0; i < 5; i++ {
			same, err := c.SameL1Chain(ctx, []eth.BlockID{{Hash: hash(95), Number: 95}})
			require.NoError(t, err)
			require.True(t, same)
		}
		require.Equal(t, 1, src.byNumberCalls)
		require.Equal(t, 1, src.finalizedCalls)
	})

	t.Run("non-finalized numbers are always fetched", func(t *testing.T) {
		now := time.Unix(1000, 0)
		src := newSource(100)
		c := newChecker(src, &now)
		for i := 0; i < 3; i++ {
			same, err := c.SameL1Chain(ctx, []eth.BlockID{{Hash: hash(105), Number: 105}})
			require.NoError(t, err)
			require.True(t, same)
		}
		require.Equal(t, 3, src.byNumberCalls)
		// Finality is refreshed at most once per interval.
		require.Equal(t, 1, src.finalizedCalls)
	})

	t.Run("mismatch on a cached finalized number is still detected", func(t *testing.T) {
		now := time.Unix(1000, 0)
		src := newSource(100)
		c := newChecker(src, &now)
		same, err := c.SameL1Chain(ctx, []eth.BlockID{{Hash: hash(95), Number: 95}})
		require.NoError(t, err)
		require.True(t, same)
		same, err = c.SameL1Chain(ctx, []eth.BlockID{{Hash: common.HexToHash("0xdead"), Number: 95}})
		require.NoError(t, err)
		require.False(t, same)
		require.Equal(t, 1, src.byNumberCalls)
	})

	t.Run("duplicate heads are checked once", func(t *testing.T) {
		now := time.Unix(1000, 0)
		src := newSource(80) // nothing finalized: no caching, only dedupe
		c := newChecker(src, &now)
		head := eth.BlockID{Hash: hash(105), Number: 105}
		same, err := c.SameL1Chain(ctx, []eth.BlockID{head, head, {}, head})
		require.NoError(t, err)
		require.True(t, same)
		require.Equal(t, 1, src.byNumberCalls)
	})

	t.Run("finality advances after the refresh interval", func(t *testing.T) {
		now := time.Unix(1000, 0)
		src := newSource(100)
		c := newChecker(src, &now)
		head := eth.BlockID{Hash: hash(105), Number: 105}
		_, err := c.SameL1Chain(ctx, []eth.BlockID{head})
		require.NoError(t, err)
		src.finalized = 110
		now = now.Add(finalizedRefreshInterval)
		_, err = c.SameL1Chain(ctx, []eth.BlockID{head}) // fetched, then cached as finalized
		require.NoError(t, err)
		_, err = c.SameL1Chain(ctx, []eth.BlockID{head}) // cache hit
		require.NoError(t, err)
		require.Equal(t, 2, src.byNumberCalls)
		require.Equal(t, 2, src.finalizedCalls)
	})

	t.Run("hash fetched before finality is not cached", func(t *testing.T) {
		now := time.Unix(1000, 0)
		src := newSource(100)
		c := newChecker(src, &now)
		// Prime finality at 100, then let it advance past 105 while 105 is fetched.
		_, err := c.SameL1Chain(ctx, []eth.BlockID{{Hash: hash(95), Number: 95}})
		require.NoError(t, err)
		head := eth.BlockID{Hash: hash(105), Number: 105}
		src.onByNumber = func() {
			src.finalized = 110
			now = now.Add(finalizedRefreshInterval)
		}
		_, err = c.SameL1Chain(ctx, []eth.BlockID{head})
		require.NoError(t, err)
		src.onByNumber = nil
		_, err = c.SameL1Chain(ctx, []eth.BlockID{head}) // must fetch again
		require.NoError(t, err)
		require.Equal(t, 3, src.byNumberCalls)
	})

	t.Run("lookup errors are not cached", func(t *testing.T) {
		now := time.Unix(1000, 0)
		src := newSource(100)
		c := newChecker(src, &now)
		_, err := c.SameL1Chain(ctx, []eth.BlockID{{Hash: hash(50), Number: 50}})
		require.Error(t, err)
		_, err = c.SameL1Chain(ctx, []eth.BlockID{{Hash: hash(50), Number: 50}})
		require.Error(t, err)
		require.Equal(t, 2, src.byNumberCalls)
	})
}
