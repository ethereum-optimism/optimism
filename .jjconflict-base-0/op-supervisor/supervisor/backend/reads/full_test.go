package reads

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

func TestReadHandles(t *testing.T) {
	newRegistry := func(t *testing.T) *Registry {
		logger := testlog.Logger(t, log.LevelInfo)
		return NewRegistry(logger)
	}
	t.Run("empty", func(t *testing.T) {
		reg := newRegistry(t)
		h := reg.AcquireHandle()
		require.True(t, h.IsValid(), "valid by default")
		h.Release()
		require.True(t, h.IsValid(), "still valid after release")
		release, err := reg.TryInvalidate(DerivedInvalidation{Timestamp: 100})
		require.NoError(t, err)
		release()
		require.True(t, h.IsValid(), "still valid after unrelated late invalidation")
	})
	t.Run("basic", func(t *testing.T) {
		reg := newRegistry(t)
		h := reg.AcquireHandle()
		require.True(t, h.IsValid(), "valid by default")
		h.DependOnDerivedTime(100)
		require.True(t, h.IsValid(), "dependency is ok")
		h.Release()
		require.True(t, h.IsValid(), "valid after release")
		release, err := reg.TryInvalidate(DerivedInvalidation{Timestamp: 10})
		require.NoError(t, err)
		release()
		require.True(t, h.IsValid(), "unaffected by invalidation after release")
	})
	t.Run("no overlap", func(t *testing.T) {
		reg := newRegistry(t)
		h := reg.AcquireHandle()
		require.True(t, h.IsValid(), "valid by default")
		h.DependOnDerivedTime(100)
		require.True(t, h.IsValid(), "dependency is ok")
		release, err := reg.TryInvalidate(DerivedInvalidation{Timestamp: 101}) // does not overlap with dependency
		require.NoError(t, err)
		release()
		require.True(t, h.IsValid(), "valid still")
		h.Release()
	})
	t.Run("invalidated single", func(t *testing.T) {
		reg := newRegistry(t)
		h := reg.AcquireHandle()
		require.True(t, h.IsValid(), "valid by default")
		h.DependOnDerivedTime(100)
		require.True(t, h.IsValid(), "dependency is ok")
		release, err := reg.TryInvalidate(DerivedInvalidation{10})
		require.NoError(t, err)
		release()
		require.False(t, h.IsValid(), "affected by invalidation before release")
		h.Release()
		require.False(t, h.IsValid(), "still considered invalid after release")
		require.ErrorIs(t, h.Err(), types.ErrInvalidatedRead, "err helper works")
	})
	t.Run("invalidated two", func(t *testing.T) {
		reg := newRegistry(t)
		h := reg.AcquireHandle()
		require.True(t, h.IsValid(), "valid by default")
		h.DependOnDerivedTime(100)
		h.DependOnDerivedTime(90)
		require.True(t, h.IsValid(), "dependency is ok")
		release, err := reg.TryInvalidate(DerivedInvalidation{10})
		require.NoError(t, err)
		release()
		h.Release()
		require.False(t, h.IsValid(), "invalidated both")
	})
	t.Run("multiple deps", func(t *testing.T) {
		reg := newRegistry(t)
		h := reg.AcquireHandle()
		require.True(t, h.IsValid(), "valid by default")
		h.DependOnDerivedTime(100)
		h.DependOnDerivedTime(90)
		require.True(t, h.IsValid(), "dependency is ok")
		release, err := reg.TryInvalidate(DerivedInvalidation{95})
		require.NoError(t, err)
		release()
		require.False(t, h.IsValid(), "expected to be invalidated")
		h.Release()
	})
	t.Run("invalidated other type", func(t *testing.T) {
		reg := newRegistry(t)
		h := reg.AcquireHandle()
		require.True(t, h.IsValid(), "valid by default")
		h.DependOnSourceBlock(100)
		release, err := reg.TryInvalidate(DerivedInvalidation{95})
		require.NoError(t, err)
		release()
		require.True(t, h.IsValid(), "depended on source 100, but did not invalidate this type")
		h.Release()
	})
	t.Run("invalidated combined", func(t *testing.T) {
		reg := newRegistry(t)
		h := reg.AcquireHandle()
		require.True(t, h.IsValid(), "valid by default")
		h.DependOnSourceBlock(2000)
		release, err := reg.TryInvalidate(InvalidationRules{
			DerivedInvalidation{Timestamp: 95},
			SourceInvalidation{Number: 1000},
		})
		require.NoError(t, err)
		release()
		require.False(t, h.IsValid())
		h.Release()
	})
	t.Run("adjust up", func(t *testing.T) {
		reg := newRegistry(t)
		h := reg.AcquireHandle()
		require.True(t, h.IsValid(), "valid by default")
		h.DependOnSourceBlock(500)
		release, err := reg.TryInvalidate(SourceInvalidation{Number: 1000})
		require.NoError(t, err)
		release()
		require.True(t, h.IsValid(), "still valid")
		h.DependOnSourceBlock(1500)
		require.False(t, h.IsValid(), "invalidated")
		h.Release()
	})
	t.Run("no concurrent invalidating", func(t *testing.T) {
		reg := newRegistry(t)
		release1, err := reg.TryInvalidate(SourceInvalidation{100})
		require.NoError(t, err)
		_, err = reg.TryInvalidate(SourceInvalidation{200})
		require.ErrorIs(t, err, types.ErrAlreadyInvalidatingRead)
		release1()
	})
	t.Run("no valid reads while invalidating", func(t *testing.T) {
		reg := newRegistry(t)
		release1, err := reg.TryInvalidate(SourceInvalidation{100})
		require.NoError(t, err)
		h := reg.AcquireHandle()
		h.DependOnSourceBlock(200)
		release1()
		require.False(t, h.IsValid(), "invalidation was ongoing when read happened")
	})
}
