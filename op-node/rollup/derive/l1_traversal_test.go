package derive

import (
	"errors"
	"math/rand"
	"testing"

	"github.com/ethereum-optimism/optimism/op-node/testlog"
	"github.com/ethereum-optimism/optimism/op-node/testutils"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

func TestL1Traversal_Step(t *testing.T) {
	rng := rand.New(rand.NewSource(1234))
	a := testutils.RandomBlockRef(rng)
	b := testutils.NextRandomRef(rng, a)
	c := testutils.NextRandomRef(rng, b)
	d := testutils.NextRandomRef(rng, c)
	e := testutils.NextRandomRef(rng, d)

	f := testutils.RandomBlockRef(rng) // a fork, doesn't build on d
	f.Number = e.Number + 1            // even though it might be the next number

	l1Fetcher := &testutils.MockL1Source{}
	l1Fetcher.ExpectL1BlockRefByNumber(b.Number, b, nil)
	// pretend there's an RPC error
	l1Fetcher.ExpectL1BlockRefByNumber(c.Number, c, errors.New("rpc error - check back later"))
	l1Fetcher.ExpectL1BlockRefByNumber(c.Number, c, nil)
	// pretend the block is not there yet for a while
	l1Fetcher.ExpectL1BlockRefByNumber(d.Number, d, ethereum.NotFound)
	l1Fetcher.ExpectL1BlockRefByNumber(d.Number, d, ethereum.NotFound)
	// it will show up though
	l1Fetcher.ExpectL1BlockRefByNumber(d.Number, d, nil)
	l1Fetcher.ExpectL1BlockRefByNumber(e.Number, e, nil)
	l1Fetcher.ExpectL1BlockRefByNumber(f.Number, f, nil)

	next := &MockOriginStage{progress: Progress{Origin: a, Closed: false}}

	tr := NewL1Traversal(testlog.Logger(t, log.LvlError), l1Fetcher, next)

	defer l1Fetcher.AssertExpectations(t)
	defer next.AssertExpectations(t)

	require.NoError(t, RepeatResetStep(t, tr.ResetStep, nil, 1))
	require.Equal(t, a, tr.Progress().Origin, "stage needs to adopt the origin of next stage on reset")
	require.False(t, tr.Progress().Closed, "stage needs to be open after reset")

	require.NoError(t, RepeatStep(t, tr.Step, Progress{}, 10))
	require.Equal(t, c, tr.Progress().Origin, "expected to be stuck on ethereum.NotFound on d")
	require.NoError(t, RepeatStep(t, tr.Step, Progress{}, 1))
	require.Equal(t, c, tr.Progress().Origin, "expected to be stuck again, should get the EOF within 1 step")
	require.ErrorIs(t, RepeatStep(t, tr.Step, Progress{}, 10), ReorgErr, "completed pipeline, until L1 input f that causes a reorg")
}
