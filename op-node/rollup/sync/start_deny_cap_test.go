package sync

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// stubL2 is a minimal L2Chain stub for cap-helper tests. Only L2BlockRefByNumber
// is exercised — the other methods panic so accidental use is loud.
type stubL2 struct {
	byNumber map[uint64]eth.L2BlockRef
	err      error
}

func newStubL2(refs ...eth.L2BlockRef) *stubL2 {
	s := &stubL2{byNumber: make(map[uint64]eth.L2BlockRef, len(refs))}
	for _, r := range refs {
		s.byNumber[r.Number] = r
	}
	return s
}

func (s *stubL2) L2BlockRefByHash(ctx context.Context, h common.Hash) (eth.L2BlockRef, error) {
	panic("stubL2.L2BlockRefByHash should not be called")
}

func (s *stubL2) L2BlockRefByLabel(ctx context.Context, label eth.BlockLabel) (eth.L2BlockRef, error) {
	panic("stubL2.L2BlockRefByLabel should not be called")
}

func (s *stubL2) L2BlockRefByNumber(ctx context.Context, number uint64) (eth.L2BlockRef, error) {
	if s.err != nil {
		return eth.L2BlockRef{}, s.err
	}
	ref, ok := s.byNumber[number]
	if !ok {
		return eth.L2BlockRef{}, fmt.Errorf("no block at number %d", number)
	}
	return ref, nil
}

// stubAuthority implements rollup.SuperAuthority for cap tests. It only
// implements CanonicalDeniedHeight non-trivially; the other methods return
// empty values.
type stubAuthority struct {
	canonicalDeniedHeight uint64
	canonicalDeniedFound  bool
	canonicalDeniedErr    error
}

func (s *stubAuthority) FullyVerifiedL2Head() (eth.BlockID, bool) { return eth.BlockID{}, false }
func (s *stubAuthority) FinalizedL2Head() (eth.BlockID, bool)     { return eth.BlockID{}, false }
func (s *stubAuthority) IsDenied(uint64, common.Hash) (bool, error) {
	return false, nil
}
func (s *stubAuthority) CanonicalDeniedHeight(_ context.Context, _ rollup.CanonicalChain) (uint64, bool, error) {
	return s.canonicalDeniedHeight, s.canonicalDeniedFound, s.canonicalDeniedErr
}

func makeRef(n uint64, hash common.Hash) eth.L2BlockRef {
	return eth.L2BlockRef{Number: n, Hash: hash}
}

func TestApplyDenyCap(t *testing.T) {
	t.Parallel()

	hashAt := func(n uint64) common.Hash {
		return common.BigToHash(common.Big1.Add(common.Big1, common.Big1.SetUint64(n)))
	}

	t.Run("nil authority is a no-op", func(t *testing.T) {
		t.Parallel()
		result := &FindHeadsResult{
			Unsafe:    makeRef(100, hashAt(100)),
			Safe:      makeRef(80, hashAt(80)),
			Finalized: makeRef(60, hashAt(60)),
		}
		before := *result
		require.NoError(t, ApplyDenyCap(context.Background(), newStubL2(), nil, result))
		require.Equal(t, before, *result)
	})

	t.Run("no canonical denied block is a no-op", func(t *testing.T) {
		t.Parallel()
		result := &FindHeadsResult{
			Unsafe:    makeRef(100, hashAt(100)),
			Safe:      makeRef(80, hashAt(80)),
			Finalized: makeRef(60, hashAt(60)),
		}
		before := *result
		auth := &stubAuthority{canonicalDeniedFound: false}
		require.NoError(t, ApplyDenyCap(context.Background(), newStubL2(), auth, result))
		require.Equal(t, before, *result)
	})

	t.Run("safe above cap is lowered; finalized below stays put", func(t *testing.T) {
		t.Parallel()
		// cap = 50, target = 49
		result := &FindHeadsResult{
			Unsafe:    makeRef(100, hashAt(100)),
			Safe:      makeRef(80, hashAt(80)),
			Finalized: makeRef(30, hashAt(30)), // below target
		}
		l2 := newStubL2(makeRef(49, hashAt(49)))
		auth := &stubAuthority{canonicalDeniedHeight: 50, canonicalDeniedFound: true}

		require.NoError(t, ApplyDenyCap(context.Background(), l2, auth, result))

		require.Equal(t, uint64(49), result.Safe.Number, "safe lowered to cap-1")
		require.Equal(t, hashAt(49), result.Safe.Hash)
		require.Equal(t, uint64(30), result.Finalized.Number, "finalized below target unchanged")
		require.Equal(t, uint64(100), result.Unsafe.Number, "unsafe must not be capped")
	})

	t.Run("safe and finalized both above cap both lowered", func(t *testing.T) {
		t.Parallel()
		result := &FindHeadsResult{
			Unsafe:    makeRef(100, hashAt(100)),
			Safe:      makeRef(80, hashAt(80)),
			Finalized: makeRef(70, hashAt(70)),
		}
		l2 := newStubL2(makeRef(49, hashAt(49)))
		auth := &stubAuthority{canonicalDeniedHeight: 50, canonicalDeniedFound: true}

		require.NoError(t, ApplyDenyCap(context.Background(), l2, auth, result))

		require.Equal(t, uint64(49), result.Safe.Number)
		require.Equal(t, uint64(49), result.Finalized.Number)
		require.Equal(t, uint64(100), result.Unsafe.Number)
	})

	t.Run("idempotent: second call on capped result is a no-op", func(t *testing.T) {
		t.Parallel()
		result := &FindHeadsResult{
			Unsafe:    makeRef(100, hashAt(100)),
			Safe:      makeRef(49, hashAt(49)),
			Finalized: makeRef(49, hashAt(49)),
		}
		before := *result
		l2 := newStubL2(makeRef(49, hashAt(49)))
		auth := &stubAuthority{canonicalDeniedHeight: 50, canonicalDeniedFound: true}

		require.NoError(t, ApplyDenyCap(context.Background(), l2, auth, result))
		require.Equal(t, before, *result, "already at target, nothing to lower")
	})

	t.Run("cap target lookup error surfaces", func(t *testing.T) {
		t.Parallel()
		result := &FindHeadsResult{
			Unsafe:    makeRef(100, hashAt(100)),
			Safe:      makeRef(80, hashAt(80)),
			Finalized: makeRef(70, hashAt(70)),
		}
		l2 := &stubL2{err: errors.New("EL unavailable")}
		auth := &stubAuthority{canonicalDeniedHeight: 50, canonicalDeniedFound: true}

		err := ApplyDenyCap(context.Background(), l2, auth, result)
		require.Error(t, err)
		require.Contains(t, err.Error(), "EL unavailable")
	})

	t.Run("authority error surfaces", func(t *testing.T) {
		t.Parallel()
		result := &FindHeadsResult{
			Unsafe:    makeRef(100, hashAt(100)),
			Safe:      makeRef(80, hashAt(80)),
			Finalized: makeRef(70, hashAt(70)),
		}
		auth := &stubAuthority{canonicalDeniedErr: errors.New("bbolt corruption")}
		err := ApplyDenyCap(context.Background(), newStubL2(), auth, result)
		require.Error(t, err)
		require.Contains(t, err.Error(), "bbolt corruption")
	})

	t.Run("cap at height 1 lowers safe to genesis (target 0)", func(t *testing.T) {
		t.Parallel()
		result := &FindHeadsResult{
			Unsafe:    makeRef(10, hashAt(10)),
			Safe:      makeRef(5, hashAt(5)),
			Finalized: makeRef(0, hashAt(0)),
		}
		l2 := newStubL2(makeRef(0, hashAt(0)))
		auth := &stubAuthority{canonicalDeniedHeight: 1, canonicalDeniedFound: true}

		require.NoError(t, ApplyDenyCap(context.Background(), l2, auth, result))
		require.Equal(t, uint64(0), result.Safe.Number)
	})

	t.Run("nil result is a no-op (defensive)", func(t *testing.T) {
		t.Parallel()
		auth := &stubAuthority{canonicalDeniedHeight: 50, canonicalDeniedFound: true}
		require.NoError(t, ApplyDenyCap(context.Background(), newStubL2(), auth, nil))
	})
}
