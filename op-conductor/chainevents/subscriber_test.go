package chainevents

import (
	"encoding/json"
	"testing"

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

func TestChainTipNumber(t *testing.T) {
	c := &chain{Blocks: nil}
	_, ok := c.tipNumber()
	require.False(t, ok, "empty chain has no tip")

	c = &chain{Blocks: map[uint64]json.RawMessage{3: nil, 7: nil, 5: nil}}
	tip, ok := c.tipNumber()
	require.True(t, ok)
	require.Equal(t, uint64(7), tip)
}
