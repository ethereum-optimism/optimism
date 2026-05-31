package chainevents

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

// newTestSubscriber builds a Subscriber wired to a cap-1 channel for frame-parsing tests.
func newTestSubscriber() (*Subscriber, chan ReorgEvent) {
	out := make(chan ReorgEvent, 1)
	return NewSubscriber(log.NewLogger(log.DiscardHandler()), "ws://test", out), out
}

// A realistic Commit frame matching the empirically-captured op-reth shape (Phase 0):
// externally-tagged variant, double-nested block.header.header, decimal-string block keys,
// and the heavy execution_outcome / trie_data fields the decoder must ignore.
const commitFrame = `{"jsonrpc":"2.0","method":"reth_subscribeChainNotifications","params":{"subscription":"0xfad3","result":{"Commit":{"new":{"blocks":{"47":{"block":{"header":{"header":{"parentHash":"0xdc04","number":"0x2f"}},"body":{"transactions":[],"ommers":[],"withdrawals":[]}},"senders":[]}},"execution_outcome":{"first_block":47},"trie_data":{}}}}}}`

func TestHandleFrameCommitIsIgnored(t *testing.T) {
	s, out := newTestSubscriber()
	s.handleFrame([]byte(commitFrame))
	select {
	case ev := <-out:
		t.Fatalf("Commit frame should yield no ReorgEvent, got %+v", ev)
	default:
	}
}

func TestHandleFrameReorgYieldsTipNumbers(t *testing.T) {
	s, out := newTestSubscriber()
	// Reorg: old chain tips at 47 (blocks 45,46,47), new chain tips at 45.
	frame := `{"jsonrpc":"2.0","method":"reth_subscribeChainNotifications","params":{"subscription":"0xabc","result":{"Reorg":{"old":{"blocks":{"45":{},"46":{},"47":{}}},"new":{"blocks":{"45":{}}}}}}}`
	s.handleFrame([]byte(frame))
	select {
	case ev := <-out:
		require.Equal(t, uint64(47), ev.OldTipNumber)
		require.Equal(t, uint64(45), ev.NewTipNumber)
		require.True(t, ev.HasNewTip)
	default:
		t.Fatal("expected a ReorgEvent for a Reorg frame")
	}
}

func TestHandleFrameReorgPureRevertHasNoNewTip(t *testing.T) {
	s, out := newTestSubscriber()
	// Pure revert: new chain is empty.
	frame := `{"jsonrpc":"2.0","method":"reth_subscribeChainNotifications","params":{"subscription":"0xabc","result":{"Reorg":{"old":{"blocks":{"46":{},"47":{}}},"new":{"blocks":{}}}}}}`
	s.handleFrame([]byte(frame))
	select {
	case ev := <-out:
		require.Equal(t, uint64(47), ev.OldTipNumber)
		require.False(t, ev.HasNewTip)
		require.Equal(t, uint64(0), ev.NewTipNumber)
	default:
		t.Fatal("expected a ReorgEvent for a pure-revert Reorg frame")
	}
}

func TestHandleFrameNonNotificationIsIgnored(t *testing.T) {
	s, out := newTestSubscriber()
	// A subscribe response (no params) must not produce an event.
	s.handleFrame([]byte(`{"jsonrpc":"2.0","id":1,"result":"0xdeadbeef"}`))
	select {
	case ev := <-out:
		t.Fatalf("subscribe response should yield no ReorgEvent, got %+v", ev)
	default:
	}
}

func TestDeliverCoalescesToLatest(t *testing.T) {
	s, out := newTestSubscriber()
	// Two events without a reader between them: the channel (cap 1) must hold the latest.
	s.deliver(ReorgEvent{NewTipNumber: 10, HasNewTip: true})
	s.deliver(ReorgEvent{NewTipNumber: 20, HasNewTip: true})
	ev := <-out
	require.Equal(t, uint64(20), ev.NewTipNumber, "cap-1 channel should coalesce to the latest event")
	select {
	case extra := <-out:
		t.Fatalf("expected only one buffered event, got extra %+v", extra)
	default:
	}
}

// subscribeResp is a minimal valid JSON-RPC subscribe reply carrying a subscription id.
const subscribeResp = `{"jsonrpc":"2.0","id":1,"result":"0x1"}`

// reorgFrameA tips old at 47, new at 45.
const reorgFrameA = `{"jsonrpc":"2.0","method":"reth_subscribeChainNotifications","params":{"subscription":"0x1","result":{"Reorg":{"old":{"blocks":{"45":{},"46":{},"47":{}}},"new":{"blocks":{"45":{}}}}}}}`

// reorgFrameB tips old at 60, new at 58.
const reorgFrameB = `{"jsonrpc":"2.0","method":"reth_subscribeChainNotifications","params":{"subscription":"0x1","result":{"Reorg":{"old":{"blocks":{"58":{},"59":{},"60":{}}},"new":{"blocks":{"58":{}}}}}}}`

// wsURLOf converts an httptest server URL to a ws:// URL.
func wsURLOf(srv *httptest.Server) string {
	return "ws" + strings.TrimPrefix(srv.URL, "http")
}

// recv reads one event from out within a generous timeout, failing the test otherwise.
func recv(t *testing.T, out <-chan ReorgEvent) ReorgEvent {
	t.Helper()
	select {
	case ev := <-out:
		return ev
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for a ReorgEvent")
		return ReorgEvent{}
	}
}

// TestSubscriberReconnectsAndResubscribes drives the full reconnect path: the first
// connection delivers a reorg then drops; the subscriber must reconnect, re-subscribe,
// emit a fresh catch-up (Resync) trigger, and resume delivering reorgs.
func TestSubscriberReconnectsAndResubscribes(t *testing.T) {
	var subscribeCount atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer func() { _ = c.CloseNow() }()
		ctx := r.Context()
		if _, _, err := c.Read(ctx); err != nil { // subscribe request
			return
		}
		n := subscribeCount.Add(1)
		if err := c.Write(ctx, websocket.MessageText, []byte(subscribeResp)); err != nil {
			return
		}
		if n == 1 {
			// Deliver a reorg, then drop the connection to force a reconnect.
			_ = c.Write(ctx, websocket.MessageText, []byte(reorgFrameA))
			time.Sleep(100 * time.Millisecond)
			return // defer CloseNow drops the socket
		}
		// Subsequent connections: deliver another reorg and stay open until teardown.
		_ = c.Write(ctx, websocket.MessageText, []byte(reorgFrameB))
		<-ctx.Done()
	}))
	defer srv.Close()

	out := make(chan ReorgEvent, 16)
	s := NewSubscriber(log.NewLogger(log.DiscardHandler()), wsURLOf(srv), out)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s.Start(ctx)

	// First subscribe → catch-up Resync, then the conn-1 reorg.
	require.True(t, recv(t, out).Resync, "first (re)subscribe must emit a Resync catch-up trigger")
	ev := recv(t, out)
	require.False(t, ev.Resync)
	require.Equal(t, uint64(47), ev.OldTipNumber)
	require.Equal(t, uint64(45), ev.NewTipNumber)

	// After the drop the subscriber reconnects: a second Resync, then the conn-2 reorg.
	require.True(t, recv(t, out).Resync, "reconnect must re-subscribe and emit another Resync")
	ev = recv(t, out)
	require.False(t, ev.Resync)
	require.Equal(t, uint64(60), ev.OldTipNumber)
	require.Equal(t, uint64(58), ev.NewTipNumber)

	require.GreaterOrEqual(t, subscribeCount.Load(), int32(2), "subscriber must have re-subscribed after the drop")
}

// TestSubscriberCommitFramesKeepReadAlive guards against filtering Commit frames at the
// transport layer: a Commit frame must be read and ignored (not reconnected on), and a
// following Reorg on the SAME connection must still be delivered.
func TestSubscriberCommitFramesKeepReadAlive(t *testing.T) {
	var subscribeCount atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer func() { _ = c.CloseNow() }()
		ctx := r.Context()
		if _, _, err := c.Read(ctx); err != nil {
			return
		}
		subscribeCount.Add(1)
		if err := c.Write(ctx, websocket.MessageText, []byte(subscribeResp)); err != nil {
			return
		}
		// Commit first (must be consumed and ignored), then a Reorg on the same socket.
		_ = c.Write(ctx, websocket.MessageText, []byte(commitFrame))
		_ = c.Write(ctx, websocket.MessageText, []byte(reorgFrameA))
		<-ctx.Done()
	}))
	defer srv.Close()

	out := make(chan ReorgEvent, 16)
	s := NewSubscriber(log.NewLogger(log.DiscardHandler()), wsURLOf(srv), out)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s.Start(ctx)

	require.True(t, recv(t, out).Resync)
	// The very next event is the Reorg — proving the Commit frame was read and skipped,
	// not turned into an event nor a reconnect.
	ev := recv(t, out)
	require.False(t, ev.Resync)
	require.Equal(t, uint64(47), ev.OldTipNumber)

	// No reconnect happened: a single subscribe handshake.
	require.Equal(t, int32(1), subscribeCount.Load(), "Commit frames must not trigger a reconnect")
}

func TestChainTipNumber(t *testing.T) {
	c := &chain{Blocks: nil}
	_, ok := c.tipNumber()
	require.False(t, ok, "empty chain has no tip")

	c = &chain{Blocks: map[uint64]json.RawMessage{3: nil, 7: nil, 5: nil}}
	tip, ok := c.tipNumber()
	require.True(t, ok)
	require.Equal(t, uint64(7), tip)
}
