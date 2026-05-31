package chainevents

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/coder/websocket"
	"github.com/ethereum/go-ethereum/log"

	opclient "github.com/ethereum-optimism/optimism/op-service/client"
)

const (
	// subscribeMethod is reth's chain-notifications subscription method.
	subscribeMethod = "reth_subscribeChainNotifications"
	// minBackoff and maxBackoff bound the reconnect backoff. Backoff is applied on
	// ALL errors (dial, subscribe, read, parse) — unlike the flashblocks handler,
	// which busy-spins on read errors and would hot-loop against an accept-then-EOF EL.
	minBackoff = 1 * time.Second
	maxBackoff = 30 * time.Second
	// readTimeout bounds a single Read while waiting for the next notification.
	readTimeout = 60 * time.Second
)

// ReorgEvent is emitted when op-reth reports a chain reorg. The reth notification
// does not carry block hashes, so only block numbers are available; the handler
// reads the authoritative head hash from the EL.
type ReorgEvent struct {
	// OldTipNumber is the highest block number of the reverted ("old") chain.
	OldTipNumber uint64
	// NewTipNumber is the highest block number of the replacement ("new") chain.
	// HasNewTip is false on a pure revert (empty "new" chain).
	NewTipNumber uint64
	HasNewTip    bool
	// Resync marks a synthetic catch-up trigger emitted on each successful (re)subscribe,
	// not a reth-reported reorg. reth_subscribeChainNotifications has no replay, so a reorg
	// that lands during a WS disconnect is lost; re-reading and committing the EL's current
	// unsafe head on reconnect reconciles any reorg missed during the gap. Idempotent: a
	// no-op when the EL head already matches the FSM head.
	Resync bool
}

// Subscriber maintains a WebSocket subscription to op-reth's reorg notifications,
// reconnecting with capped backoff, and delivers ReorgEvents on out.
//
// out is a cap-1 channel shared with the consumer. Delivery is non-blocking and
// coalesces to the latest event (see deliver): the read loop never blocks on a
// busy consumer, so reth's tokio::broadcast lagged-receiver drop is not tripped
// while the consumer is mid-fetch. A coalesced/stale event is safe because the
// handler re-reads the EL's current head before committing.
type Subscriber struct {
	log   log.Logger
	wsURL string
	out   chan ReorgEvent
}

// NewSubscriber creates a Subscriber that delivers reorg events on out (cap 1).
func NewSubscriber(logger log.Logger, wsURL string, out chan ReorgEvent) *Subscriber {
	return &Subscriber{log: logger, wsURL: wsURL, out: out}
}

// Start runs the subscription loop on its own goroutine until ctx is cancelled.
func (s *Subscriber) Start(ctx context.Context) {
	go s.run(ctx)
}

func (s *Subscriber) run(ctx context.Context) {
	backoff := minBackoff
	for {
		if ctx.Err() != nil {
			return
		}
		subscribed, err := s.connectAndRead(ctx)
		if subscribed {
			// A live connection resets the backoff so a later drop reconnects fast.
			backoff = minBackoff
		}
		if err == nil {
			// connectAndRead only returns nil error on ctx cancellation.
			return
		}
		if ctx.Err() != nil {
			return
		}
		s.log.Warn("reth reorg subscription error; reconnecting", "err", err, "backoff", backoff)
		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}
		backoff = nextBackoff(backoff)
	}
}

func nextBackoff(d time.Duration) time.Duration {
	d *= 2
	if d > maxBackoff {
		return maxBackoff
	}
	return d
}

// connectAndRead dials, subscribes, and reads notifications until an error or ctx
// cancellation. subscribed reports whether the subscribe handshake succeeded (used
// by run to reset backoff). It returns a nil error only when ctx is cancelled.
func (s *Subscriber) connectAndRead(ctx context.Context) (subscribed bool, err error) {
	conn, err := opclient.DialWS(ctx, opclient.WSConfig{
		URL:         s.wsURL,
		MaxAttempts: 1,
		Log:         s.log,
	})
	if err != nil {
		return false, fmt.Errorf("dial: %w", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "closing reorg subscription")

	req := fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"method":%q,"params":[]}`, subscribeMethod)
	if err := conn.Write(ctx, websocket.MessageText, []byte(req)); err != nil {
		return false, fmt.Errorf("write subscribe request: %w", err)
	}
	if err := s.readSubscribeResponse(ctx, conn); err != nil {
		return false, err
	}
	s.log.Info("subscribed to reth chain notifications", "url", s.wsURL)
	// Emit a catch-up trigger on every (re)subscribe so the leader reconciles its EL's
	// current unsafe head, recovering any reorg missed while the WS was disconnected
	// (reth notifications have no replay). Idempotent on the handler side.
	s.deliver(ReorgEvent{Resync: true})

	for {
		if ctx.Err() != nil {
			return true, nil
		}
		readCtx, cancel := context.WithTimeout(ctx, readTimeout)
		_, data, readErr := conn.Read(readCtx)
		cancel()
		if readErr != nil {
			return true, fmt.Errorf("read notification: %w", readErr)
		}
		s.handleFrame(data)
	}
}

func (s *Subscriber) readSubscribeResponse(ctx context.Context, conn *opclient.WSClient) error {
	readCtx, cancel := context.WithTimeout(ctx, readTimeout)
	defer cancel()
	_, data, err := conn.Read(readCtx)
	if err != nil {
		return fmt.Errorf("read subscribe response: %w", err)
	}
	var resp subscribeResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return fmt.Errorf("decode subscribe response: %w", err)
	}
	if resp.Error != nil {
		return fmt.Errorf("subscribe rejected: %s", string(*resp.Error))
	}
	if resp.Result == "" {
		return fmt.Errorf("subscribe response missing subscription id: %s", string(data))
	}
	return nil
}

// handleFrame parses a single WS frame and, on a Reorg notification, delivers a
// ReorgEvent. Commit notifications (forward sealed blocks) are ignored: those are
// already covered by op-node's existing forward CommitUnsafePayload writes.
func (s *Subscriber) handleFrame(data []byte) {
	var env notificationEnvelope
	if err := json.Unmarshal(data, &env); err != nil {
		s.log.Warn("failed to decode reth notification envelope", "err", err)
		return
	}
	if len(env.Params.Result) == 0 {
		return // not a subscription notification
	}
	var notif canonStateNotification
	if err := json.Unmarshal(env.Params.Result, &notif); err != nil {
		s.log.Warn("failed to decode chain notification", "err", err)
		return
	}
	if notif.Reorg == nil {
		return // Commit (forward) variant
	}
	ev := ReorgEvent{}
	if n, ok := notif.Reorg.Old.tipNumber(); ok {
		ev.OldTipNumber = n
	}
	if n, ok := notif.Reorg.New.tipNumber(); ok {
		ev.NewTipNumber = n
		ev.HasNewTip = true
	}
	s.deliver(ev)
}

// deliver performs a non-blocking, best-effort-latest send on the cap-1 channel.
// A bare drop on a cap-1 channel is not latest-wins, so on a full channel we drain
// the stale buffered event and store the newest. If the consumer reads concurrently
// between drain and re-send, the channel may momentarily hold a slightly older event
// — the guarantee is "best-effort latest, made safe by handler-side re-validation".
func (s *Subscriber) deliver(ev ReorgEvent) {
	select {
	case s.out <- ev:
	default:
		select {
		case <-s.out:
		default:
		}
		select {
		case s.out <- ev:
		default:
		}
	}
}
