package sysgo

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// fakeRpcCaller runs one scripted behavior per admin_peers call; the last
// behavior repeats for any further calls.
type fakeRpcCaller struct {
	calls     int
	behaviors []func(ctx context.Context) ([]peer, error)
}

func (f *fakeRpcCaller) CallContext(ctx context.Context, result interface{}, method string, args ...interface{}) error {
	i := f.calls
	if i >= len(f.behaviors) {
		i = len(f.behaviors) - 1
	}
	f.calls++
	peers, err := f.behaviors[i](ctx)
	if err != nil {
		return err
	}
	raw, err := json.Marshal(peers)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, result)
}

// blockUntilAttemptDeadline waits out the per-attempt deadline, then returns
// the given error — simulating an RPC call that outlives its attempt budget.
func blockUntilAttemptDeadline(err error) func(ctx context.Context) ([]peer, error) {
	return func(ctx context.Context) ([]peer, error) {
		<-ctx.Done()
		return nil, err
	}
}

func hasPeer(id string) func([]peer) bool {
	return func(peers []peer) bool {
		for _, p := range peers {
			if p.ID == id {
				return true
			}
		}
		return false
	}
}

func TestPollPeerListRetriesAttemptTimeout(t *testing.T) {
	node := &fakeRpcCaller{behaviors: []func(ctx context.Context) ([]peer, error){
		blockUntilAttemptDeadline(context.DeadlineExceeded),
		func(ctx context.Context) ([]peer, error) { return []peer{{ID: "abc"}}, nil },
	}}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := pollPeerList(ctx, node, 10*time.Millisecond, 50*time.Millisecond, hasPeer("abc"))
	require.NoError(t, err)
	require.Equal(t, 2, node.calls, "the attempt timeout must be retried")
}

func TestPollPeerListFailsFastOnRPCError(t *testing.T) {
	// The RPC error returns after the attempt deadline has already expired:
	// an expired attempt context alone must not turn it into a retry.
	rpcErr := errors.New("the method admin_peers does not exist")
	node := &fakeRpcCaller{behaviors: []func(ctx context.Context) ([]peer, error){
		blockUntilAttemptDeadline(rpcErr),
	}}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := pollPeerList(ctx, node, 10*time.Millisecond, 50*time.Millisecond, hasPeer("abc"))
	require.ErrorIs(t, err, rpcErr)
	require.NotErrorIs(t, err, context.DeadlineExceeded)
	require.Equal(t, 1, node.calls, "a non-timeout error must not be retried")
}

func TestPollPeerListPermanentErrorAfterTimeoutIsNotWrapped(t *testing.T) {
	// A retained attempt timeout must not be attached to a later permanent
	// error: the terminal result would otherwise classify as a timeout.
	rpcErr := errors.New("connection refused")
	node := &fakeRpcCaller{behaviors: []func(ctx context.Context) ([]peer, error){
		blockUntilAttemptDeadline(context.DeadlineExceeded),
		func(ctx context.Context) ([]peer, error) { return nil, rpcErr },
	}}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := pollPeerList(ctx, node, 10*time.Millisecond, 50*time.Millisecond, hasPeer("abc"))
	require.ErrorIs(t, err, rpcErr)
	require.NotErrorIs(t, err, context.DeadlineExceeded)
	require.Equal(t, 2, node.calls)
}

func TestPollPeerListOuterBudgetExpiry(t *testing.T) {
	node := &fakeRpcCaller{behaviors: []func(ctx context.Context) ([]peer, error){
		blockUntilAttemptDeadline(context.DeadlineExceeded),
	}}
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	err := pollPeerList(ctx, node, 10*time.Millisecond, 50*time.Millisecond, hasPeer("abc"))
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.ErrorContains(t, err, "last admin_peers attempt error")
	require.GreaterOrEqual(t, node.calls, 2, "attempt timeouts must be retried until the outer budget expires")
}
