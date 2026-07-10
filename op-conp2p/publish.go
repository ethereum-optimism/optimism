package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/gorilla/websocket"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/p2p"
)

// The publish path closes the sidecar loop for a SEQUENCING node: the node
// signs + canonically encodes each unsafe block it builds and serves it over
// the Direct Sync websocket; the sidecar holds a CURSOR subscription to that
// feed and re-publishes each payload's bytes VERBATIM to the OP gossipsub
// /blocks topics (PublishRawSignedL2Payload — no re-encode, so byte fidelity
// with what the node signed holds by construction).
//
// Loss handling: the sidecar tracks the last published block and passes
// `fromBlock = last + 1` on every (re)subscribe, so the producer's replay ring
// heals disconnects and in-stream gaps (a detected gap forces a resubscribe,
// which replays the hole). A cursor that fell below the producer's ring
// (-32020) downgrades once to a live subscribe — those blocks are gone from
// gossip's perspective and verifiers recover them via req-resp or L1.
//
// The signer symmetry mirrors the receive path: the NODE owns the key; the
// sidecar holds zero signer state and never re-signs.

// Direct Sync method names served by the node's websocket
// (crates/lib/opql-direct-sync/src/ws_server.rs).
const (
	subscribeUnsafePayloadsMethod = "opql_subscribeUnsafePayloads"
	unsafePayloadNotifMethod      = "opql_unsafePayload"
)

// errBelowHorizon is the producer's below-horizon rejection code: the cursor
// predates its signed replay ring.
const errBelowHorizon = -32020

// publishReconnectBackoff paces reconnect attempts after the feed drops. We
// reconnect forever (a live sidecar must survive sequencer restarts).
const publishReconnectBackoff = 2 * time.Second

// payloadPublisher bridges the node's Direct Sync feed onto the OP gossip
// /blocks topics.
type payloadPublisher struct {
	log log.Logger
	url string
	out p2p.GossipOut

	// lastBlock is the highest block published to gossip (0 = none yet); the
	// resubscribe cursor is lastBlock + 1.
	lastBlock uint64
	// liveOnly is set after a below-horizon rejection: the missed range cannot
	// be replayed, so the next subscribe joins live.
	liveOnly bool
}

// run holds the cursor subscription and publishes every signed payload,
// reconnecting forever on drops. Returns when ctx is done.
func (p *payloadPublisher) run(ctx context.Context) {
	for {
		err := p.consumeFeed(ctx)
		if ctx.Err() != nil {
			return
		}
		p.log.Warn("direct-sync feed dropped; reconnecting from cursor",
			"url", p.url, "cursor", p.cursor(), "err", err)
		select {
		case <-ctx.Done():
			return
		case <-time.After(publishReconnectBackoff):
		}
	}
}

func (p *payloadPublisher) cursor() uint64 {
	if p.lastBlock == 0 || p.liveOnly {
		return 0 // live join
	}
	return p.lastBlock + 1
}

// consumeFeed dials the websocket, subscribes from the cursor, and publishes
// payloads until the connection fails or ctx is done. A detected in-stream gap
// returns an error on purpose: the reconnect resubscribes from the cursor and
// the producer's ring replays the hole.
func (p *payloadPublisher) consumeFeed(ctx context.Context) error {
	dialCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	conn, _, err := websocket.DefaultDialer.DialContext(dialCtx, p.url, nil)
	cancel()
	if err != nil {
		return fmt.Errorf("failed to dial direct-sync ws %q: %w", p.url, err)
	}
	defer conn.Close()

	// Tie the blocking reads to ctx: close the connection on cancellation so
	// ReadMessage returns.
	stop := context.AfterFunc(ctx, func() { _ = conn.Close() })
	defer stop()

	if err := subscribeUnsafePayloads(conn, p.cursor()); err != nil {
		var below *belowHorizonError
		if !errors.As(err, &below) {
			return err
		}
		p.log.Warn("cursor fell below the producer's signed ring; joining live "+
			"(verifiers recover the gap via req-resp or L1)",
			"cursor", p.cursor(), "oldestSigned", below.oldestSigned)
		p.liveOnly = true
		if err := subscribeUnsafePayloads(conn, p.cursor()); err != nil {
			return err
		}
	}
	p.liveOnly = false
	p.log.Info("subscribed to direct-sync feed", "url", p.url, "nextCursor", p.cursor())

	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			return fmt.Errorf("direct-sync ws read failed: %w", err)
		}
		decoded, ok := decodeUnsafePayloadNotification(p.log, data)
		if !ok {
			continue
		}
		if gap := p.publish(ctx, decoded); gap {
			// Force a resubscribe from the cursor: the ring replays the hole.
			return fmt.Errorf("direct-sync feed gap at block %d (cursor %d); resubscribing",
				decoded.blockNumber(), p.cursor())
		}
	}
}

// belowHorizonError carries the producer's -32020 rejection.
type belowHorizonError struct {
	oldestSigned string
}

func (e *belowHorizonError) Error() string {
	return fmt.Sprintf("cursor below signed horizon (oldest %s)", e.oldestSigned)
}

// subscribeUnsafePayloads performs the JSON-RPC subscription handshake,
// passing the cursor when non-zero. The feed is served by jsonrpsee with
// explicit method names, so this speaks the raw subscription protocol rather
// than geth's namespace convention.
func subscribeUnsafePayloads(conn *websocket.Conn, fromBlock uint64) error {
	params := []any{}
	if fromBlock > 0 {
		params = append(params, map[string]any{"fromBlock": hexutil.EncodeUint64(fromBlock)})
	}
	req := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  subscribeUnsafePayloadsMethod,
		"params":  params,
	}
	if err := conn.WriteJSON(req); err != nil {
		return fmt.Errorf("failed to send subscribe request: %w", err)
	}
	// The subscription confirmation is the response to id 1. The server does not
	// send notifications before confirming, so the next message is the response.
	var resp struct {
		ID     json.RawMessage `json:"id"`
		Result json.RawMessage `json:"result"`
		Error  *struct {
			Code    int             `json:"code"`
			Message string          `json:"message"`
			Data    json.RawMessage `json:"data"`
		} `json:"error"`
	}
	if err := conn.ReadJSON(&resp); err != nil {
		return fmt.Errorf("failed to read subscribe response: %w", err)
	}
	if resp.Error != nil {
		if resp.Error.Code == errBelowHorizon {
			var data struct {
				OldestSigned string `json:"oldestSigned"`
			}
			_ = json.Unmarshal(resp.Error.Data, &data)
			return &belowHorizonError{oldestSigned: data.OldestSigned}
		}
		return fmt.Errorf("subscribe request rejected: %s (code %d)", resp.Error.Message, resp.Error.Code)
	}
	if len(resp.Result) == 0 {
		return errors.New("subscribe response carried no subscription id")
	}
	return nil
}

// decodeUnsafePayloadNotification parses one websocket message into a
// publishable decoded payload. Non-notification frames, undecodable payloads,
// and UNSIGNED (cold-tier) messages are logged and skipped — an unsigned
// payload carries no gossip-valid signature and must never be published.
func decodeUnsafePayloadNotification(logger log.Logger, data []byte) (*decodedDirectSyncPayload, bool) {
	var notif struct {
		Method string `json:"method"`
		Params struct {
			Result json.RawMessage `json:"result"`
		} `json:"params"`
	}
	if err := json.Unmarshal(data, &notif); err != nil {
		logger.Warn("undecodable direct-sync ws frame; skipping", "err", err)
		return nil, false
	}
	if notif.Method != unsafePayloadNotifMethod {
		return nil, false
	}
	var msg directSyncMsg
	if err := json.Unmarshal(notif.Params.Result, &msg); err != nil {
		logger.Warn("undecodable direct-sync message; skipping", "err", err)
		return nil, false
	}
	if !msg.Signed {
		logger.Warn("direct-sync feed delivered an unsigned payload; not gossiping")
		return nil, false
	}
	decoded, err := decodeDirectSyncPayload(&msg)
	if err != nil {
		logger.Error("direct-sync payload failed decode; skipping", "err", err)
		return nil, false
	}
	return decoded, true
}

// publish re-publishes one node-signed payload's bytes verbatim to the
// versioned /blocks topic. Returns gap=true when the block does not extend the
// published sequence (the caller resubscribes from the cursor to replay it).
func (p *payloadPublisher) publish(ctx context.Context, decoded *decodedDirectSyncPayload) (gap bool) {
	block := decoded.blockNumber()
	if p.lastBlock != 0 && block > p.lastBlock+1 {
		return true
	}

	if err := p.out.PublishRawSignedL2Payload(ctx, decoded.timestamp(), decoded.raw); err != nil {
		p.log.Error("failed to publish signed unsafe payload to gossip",
			"block", block, "hash", decoded.envelope.ExecutionPayload.BlockHash, "err", err)
		return false
	}
	if block > p.lastBlock {
		p.lastBlock = block
	}
	p.log.Info("published signed unsafe payload to gossip",
		"block", block, "hash", decoded.envelope.ExecutionPayload.BlockHash)
	return false
}
