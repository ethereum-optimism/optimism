package engine

import (
	"context"
	"errors"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
)

func TestCrossSafeCache_EmptyReturnsFalse(t *testing.T) {
	c := newCrossSafeCache(testlog.Logger(t, 0))
	_, ok := c.Get(context.Background(), &testutils.MockEngine{}, eth.L2BlockRef{Number: 100})
	require.False(t, ok)
}

func TestCrossSafeCache_CanonicalReturnsCached(t *testing.T) {
	cached := eth.L2BlockRef{Hash: common.Hash{0xaa}, Number: 80}
	engine := &testutils.MockEngine{}
	engine.ExpectL2BlockRefByNumber(cached.Number, cached, nil)

	c := newCrossSafeCache(testlog.Logger(t, 0))
	c.Store(cached)

	got, ok := c.Get(context.Background(), engine, eth.L2BlockRef{Number: 100})
	require.True(t, ok)
	require.Equal(t, cached, got)
}

func TestCrossSafeCache_NonCanonicalClears(t *testing.T) {
	cached := eth.L2BlockRef{Hash: common.Hash{0xaa}, Number: 80}
	differentCanonical := eth.L2BlockRef{Hash: common.Hash{0xbb}, Number: 80}
	engine := &testutils.MockEngine{}
	engine.ExpectL2BlockRefByNumber(cached.Number, differentCanonical, nil)

	c := newCrossSafeCache(testlog.Logger(t, 0))
	c.Store(cached)

	_, ok := c.Get(context.Background(), engine, eth.L2BlockRef{Number: 100})
	require.False(t, ok)

	// Second call short-circuits: cache cleared, no further engine calls expected.
	_, ok = c.Get(context.Background(), engine, eth.L2BlockRef{Number: 100})
	require.False(t, ok)
}

func TestCrossSafeCache_AheadOfLocalSafeClears(t *testing.T) {
	cached := eth.L2BlockRef{Hash: common.Hash{0xaa}, Number: 200}
	engine := &testutils.MockEngine{}

	c := newCrossSafeCache(testlog.Logger(t, 0))
	c.Store(cached)

	_, ok := c.Get(context.Background(), engine, eth.L2BlockRef{Number: 100})
	require.False(t, ok, "cache ahead of local-safe must not be returned")

	// And it's been cleared — no engine call ever happened for that read, and the
	// next read short-circuits on empty.
	_, ok = c.Get(context.Background(), engine, eth.L2BlockRef{Number: 100})
	require.False(t, ok)
}

func TestCrossSafeCache_EngineErrorClears(t *testing.T) {
	cached := eth.L2BlockRef{Hash: common.Hash{0xaa}, Number: 80}
	engine := &testutils.MockEngine{}
	engine.ExpectL2BlockRefByNumber(cached.Number, eth.L2BlockRef{}, errors.New("boom"))

	c := newCrossSafeCache(testlog.Logger(t, 0))
	c.Store(cached)

	_, ok := c.Get(context.Background(), engine, eth.L2BlockRef{Number: 100})
	require.False(t, ok)

	// Cache cleared on lookup failure; next read short-circuits.
	_, ok = c.Get(context.Background(), engine, eth.L2BlockRef{Number: 100})
	require.False(t, ok)
}
