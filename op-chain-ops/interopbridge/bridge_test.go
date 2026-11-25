package interopbridge

import (
	"context"
	"errors"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// stubBlockRefs replies to head lookups with the next scripted result.
type stubBlockRefs struct {
	results []stubHead
	calls   int
}

type stubHead struct {
	time uint64
	err  error
}

func (s *stubBlockRefs) BlockRefByLabel(ctx context.Context, label eth.BlockLabel) (eth.BlockRef, error) {
	next := s.results[min(s.calls, len(s.results)-1)]
	s.calls++
	if next.err != nil {
		return eth.BlockRef{}, next.err
	}
	return eth.BlockRef{Number: uint64(s.calls), Time: next.time}, nil
}

func (s *stubBlockRefs) BlockRefByNumber(ctx context.Context, num uint64) (eth.BlockRef, error) {
	panic("not used")
}

func (s *stubBlockRefs) BlockRefByHash(ctx context.Context, hash common.Hash) (eth.BlockRef, error) {
	panic("not used")
}

func TestAwaitTime(t *testing.T) {
	t.Run("returns once the head reaches the time", func(t *testing.T) {
		cl := &stubBlockRefs{results: []stubHead{{time: 99}, {time: 100}}}
		require.NoError(t, awaitTime(cl)(context.Background(), 100))
		require.Equal(t, 2, cl.calls)
	})

	t.Run("keeps waiting after a lookup failure", func(t *testing.T) {
		cl := &stubBlockRefs{results: []stubHead{{err: errors.New("no reply")}, {time: 100}}}
		require.NoError(t, awaitTime(cl)(context.Background(), 100))
		require.Equal(t, 2, cl.calls)
	})

	t.Run("reports the head it was still waiting on", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		err := awaitTime(&stubBlockRefs{results: []stubHead{{time: 99}}})(ctx, 100)
		require.ErrorIs(t, err, context.Canceled)
		require.ErrorContains(t, err, "timestamp 99")
	})

	t.Run("reports the lookup failure it was still retrying", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		lookupErr := errors.New("no reply")
		err := awaitTime(&stubBlockRefs{results: []stubHead{{err: lookupErr}}})(ctx, 100)
		require.ErrorIs(t, err, lookupErr)
		require.ErrorIs(t, err, context.Canceled)
	})
}
