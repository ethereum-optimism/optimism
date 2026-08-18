package sysgo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"

	"github.com/ethereum-optimism/optimism/op-service/testreq"
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

func TestPollPeerListMidPollDeadlineWithLiveOuterCtx(t *testing.T) {
	// After a retained attempt timeout, a later CallContext can return
	// context.DeadlineExceeded while both the attempt and outer contexts are
	// still live (e.g. a deadline internal to the RPC client). That is not an
	// attempt timeout, so it must fail fast — and the stale retained timeout
	// must not be attached, since the outer budget did not expire.
	staleTimeout := fmt.Errorf("first attempt stalled: %w", context.DeadlineExceeded)
	node := &fakeRpcCaller{behaviors: []func(ctx context.Context) ([]peer, error){
		blockUntilAttemptDeadline(staleTimeout),
		func(ctx context.Context) ([]peer, error) { return nil, context.DeadlineExceeded },
	}}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := pollPeerList(ctx, node, 10*time.Millisecond, 50*time.Millisecond, hasPeer("abc"))
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.NotContains(t, err.Error(), "last admin_peers attempt error",
		"a mid-poll deadline error with a live outer budget must not carry the stale retained timeout")
	require.Equal(t, 2, node.calls, "a deadline error from a live attempt context must not be retried")
}

// adminRpcCaller fakes a node's admin RPC surface and appends
// "<name>:<method>" to a shared call log on every call.
type adminRpcCaller struct {
	name     string
	log      *[]string
	nodeInfo p2p.NodeInfo
	peers    func() []peer
}

func (a *adminRpcCaller) CallContext(ctx context.Context, result interface{}, method string, args ...interface{}) error {
	*a.log = append(*a.log, a.name+":"+method)
	switch method {
	case "admin_nodeInfo":
		*(result.(*p2p.NodeInfo)) = a.nodeInfo
	case "admin_addPeer", "admin_addTrustedPeer":
		*(result.(*bool)) = true
	case "admin_peers":
		*(result.(*[]peer)) = a.peers()
	default:
		return fmt.Errorf("unexpected method %q", method)
	}
	return nil
}

// TestConnectP2PTrustsAcceptorOnlyAfterSessionEstablished pins the dial
// sequencing: the acceptor's trusted entry (which makes reth dial back) must
// not be added until the initiator's dial shows up in admin_peers.
func TestConnectP2PTrustsAcceptorOnlyAfterSessionEstablished(t *testing.T) {
	t.Setenv("SKIP_P2P_CONNECTION_CHECK", "")

	newNode := func(port int) *enode.Node {
		key, err := crypto.GenerateKey()
		require.NoError(t, err)
		return enode.NewV4(&key.PublicKey, net.ParseIP("127.0.0.1"), port, port)
	}
	acceptorNode, initiatorNode := newNode(30303), newNode(30304)

	var callLog []string
	polls := 0
	initiator := &adminRpcCaller{
		name: "initiator",
		log:  &callLog,
		nodeInfo: p2p.NodeInfo{
			ID:    initiatorNode.ID().String(),
			Enode: initiatorNode.URLv4(),
		},
		peers: func() []peer {
			// The session is not up on the first poll.
			polls++
			if polls == 1 {
				return nil
			}
			return []peer{{ID: acceptorNode.ID().String()}}
		},
	}
	acceptor := &adminRpcCaller{
		name: "acceptor",
		log:  &callLog,
		nodeInfo: p2p.NodeInfo{
			ID:    acceptorNode.ID().String(),
			Enode: acceptorNode.URLv4(),
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	ConnectP2P(ctx, testreq.New(t), initiator, acceptor)

	trustedIdx := slices.Index(callLog, "acceptor:admin_addTrustedPeer")
	require.NotEqual(t, -1, trustedIdx, "acceptor must be marked trusted")
	pollsBeforeTrusted := 0
	for _, call := range callLog[:trustedIdx] {
		if call == "initiator:admin_peers" {
			pollsBeforeTrusted++
		}
	}
	require.GreaterOrEqual(t, pollsBeforeTrusted, 2,
		"acceptor trusted entry must be added only after the initiator's dial is observed connected: %v", callLog)
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
