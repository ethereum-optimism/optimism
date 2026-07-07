package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/gorilla/websocket"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/p2p"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	opsigner "github.com/ethereum-optimism/optimism/op-service/signer"
)

// The publish path closes the sidecar loop for a SEQUENCING op-con-node: the
// node signs each unsafe block it builds and multicasts the signed envelope
// over a websocket (opql_subscribeSignedUnsafePayloads); the sidecar subscribes
// to that feed and re-publishes each envelope to the OP gossipsub /blocks
// topics, exactly as op-node's sequencer would.
//
// The signer symmetry mirrors the receive path: the NODE owns the key and the
// signature; the sidecar holds zero signer state and only re-encodes bytes. The
// block is published WITH the node-provided signature (PublishSignedL2Payload),
// never re-signed here.
//
// Delivery is best-effort/live-forward, matching the feed's semantics: on a
// dropped connection the sidecar reconnects with backoff and resumes from the
// next live block; a gap means dropped blocks (verifiers fill them via L1
// derivation or req-resp sync, not via the sidecar).

// Subscription method triple served by op-con-node's signed-payload websocket
// (crates/nodes/op-con-node/src/runtime/signed_payload_ws.rs).
const (
	subscribeSignedPayloadsMethod = "opql_subscribeSignedUnsafePayloads"
	signedPayloadNotifMethod      = "opql_signedUnsafePayload"
)

// publishReconnectBackoff paces reconnect attempts after the feed drops. The
// feed is best-effort, so we reconnect forever (a live sidecar must survive
// sequencer restarts).
const publishReconnectBackoff = 2 * time.Second

// signedPayloadMsg is op-con-node's signed-envelope wire shape — the SAME shape
// admin_verifyUnsafePayload consumes, so the fields mirror buildVerifyRequest's
// output in reverse.
type signedPayloadMsg struct {
	ExecutionPayload      *eth.ExecutionPayload `json:"executionPayload"`
	ParentBeaconBlockRoot *common.Hash          `json:"parentBeaconBlockRoot"`
	Signature             hexutil.Bytes         `json:"signature"`
	PayloadHash           common.Hash           `json:"payloadHash"`
}

// toSignedEnvelope validates the message and converts it to the pre-signed
// publish type. The node-supplied payloadHash is checked against a local
// re-encode of the payload (keccak of the exact bytes the gossip message will
// carry): a mismatch means the re-encode diverged from what the node signed, so
// publishing would produce a block every verifier rejects — fail loudly instead.
func (m *signedPayloadMsg) toSignedEnvelope() (*opsigner.SignedExecutionPayloadEnvelope, error) {
	if m.ExecutionPayload == nil {
		return nil, errors.New("missing executionPayload")
	}
	if len(m.Signature) != 65 {
		return nil, fmt.Errorf("signature must be 65 bytes, got %d", len(m.Signature))
	}
	envelope := &eth.ExecutionPayloadEnvelope{
		ExecutionPayload:      m.ExecutionPayload,
		ParentBeaconBlockRoot: m.ParentBeaconBlockRoot,
	}

	// Re-encode exactly as PublishSignedL2Payload will (envelope SSZ when a
	// parent beacon root is present, bare payload SSZ otherwise) and check the
	// signed hash covers those bytes.
	var buf bytes.Buffer
	if envelope.ParentBeaconBlockRoot != nil {
		if _, err := envelope.MarshalSSZ(&buf); err != nil {
			return nil, fmt.Errorf("failed to SSZ-encode payload envelope: %w", err)
		}
	} else {
		if _, err := envelope.ExecutionPayload.MarshalSSZ(&buf); err != nil {
			return nil, fmt.Errorf("failed to SSZ-encode payload: %w", err)
		}
	}
	if got := opsigner.PayloadHash(buf.Bytes()); got != m.PayloadHash {
		return nil, fmt.Errorf("payload hash mismatch: node signed %s but local re-encode hashes to %s", m.PayloadHash, got)
	}

	signed := &opsigner.SignedExecutionPayloadEnvelope{Envelope: envelope}
	copy(signed.Signature[:], m.Signature)
	// OP gossip carries the raw recovery id (0/1) in the v byte; verifiers
	// reject the Ethereum-legacy 27/28 encoding outright. Normalizing here does
	// not alter the signed content — v is recovery metadata, not part of the
	// signed message — so a feed using the legacy encoding still publishes as
	// valid gossip.
	if v := signed.Signature[64]; v == 27 || v == 28 {
		signed.Signature[64] = v - 27
	}
	return signed, nil
}

// payloadPublisher bridges the node's signed-payload websocket feed onto the OP
// gossip /blocks topics.
type payloadPublisher struct {
	log log.Logger
	url string
	out p2p.GossipOut

	// lastBlock tracks feed contiguity for gap warnings (0 = none seen yet).
	lastBlock uint64
}

// run subscribes to the signed-payload feed and publishes every received
// envelope, reconnecting forever on drops. Returns when ctx is done.
func (p *payloadPublisher) run(ctx context.Context) {
	for {
		err := p.consumeFeed(ctx)
		if ctx.Err() != nil {
			return
		}
		p.log.Warn("signed-payload feed dropped; reconnecting", "url", p.url, "err", err)
		select {
		case <-ctx.Done():
			return
		case <-time.After(publishReconnectBackoff):
		}
	}
}

// consumeFeed dials the websocket, subscribes, and publishes envelopes until
// the connection fails or ctx is done.
func (p *payloadPublisher) consumeFeed(ctx context.Context) error {
	dialCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	conn, _, err := websocket.DefaultDialer.DialContext(dialCtx, p.url, nil)
	cancel()
	if err != nil {
		return fmt.Errorf("failed to dial signed-payload ws %q: %w", p.url, err)
	}
	defer conn.Close()

	// Tie the blocking reads to ctx: close the connection on cancellation so
	// ReadMessage returns.
	stop := context.AfterFunc(ctx, func() { _ = conn.Close() })
	defer stop()

	if err := subscribeSignedPayloads(conn); err != nil {
		return err
	}
	p.log.Info("subscribed to signed-payload feed", "url", p.url)

	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			return fmt.Errorf("signed-payload ws read failed: %w", err)
		}
		envelope, ok := decodeSignedPayloadNotification(p.log, data)
		if !ok {
			continue
		}
		p.publish(ctx, envelope)
	}
}

// subscribeSignedPayloads performs the JSON-RPC subscription handshake. The
// feed is served by jsonrpsee with explicit method names, so this speaks the
// raw subscription protocol rather than geth's namespace convention.
func subscribeSignedPayloads(conn *websocket.Conn) error {
	req := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  subscribeSignedPayloadsMethod,
		"params":  []any{},
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
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := conn.ReadJSON(&resp); err != nil {
		return fmt.Errorf("failed to read subscribe response: %w", err)
	}
	if resp.Error != nil {
		return fmt.Errorf("subscribe request rejected: %s (code %d)", resp.Error.Message, resp.Error.Code)
	}
	if len(resp.Result) == 0 {
		return errors.New("subscribe response carried no subscription id")
	}
	return nil
}

// decodeSignedPayloadNotification parses one websocket message into a publishable
// envelope. Non-notification frames and undecodable payloads are logged and
// skipped (best-effort feed; a bad message must not kill the subscription).
func decodeSignedPayloadNotification(logger log.Logger, data []byte) (*opsigner.SignedExecutionPayloadEnvelope, bool) {
	var notif struct {
		Method string `json:"method"`
		Params struct {
			Result json.RawMessage `json:"result"`
		} `json:"params"`
	}
	if err := json.Unmarshal(data, &notif); err != nil {
		logger.Warn("undecodable signed-payload ws frame; skipping", "err", err)
		return nil, false
	}
	if notif.Method != signedPayloadNotifMethod {
		return nil, false
	}
	var msg signedPayloadMsg
	if err := json.Unmarshal(notif.Params.Result, &msg); err != nil {
		logger.Warn("undecodable signed-payload envelope; skipping", "err", err)
		return nil, false
	}
	envelope, err := msg.toSignedEnvelope()
	if err != nil {
		logger.Error("signed-payload envelope failed publish validation; skipping", "err", err)
		return nil, false
	}
	return envelope, true
}

// publish re-publishes one node-signed envelope to the versioned /blocks topic
// (fork selection by payload timestamp, inside PublishSignedL2Payload).
func (p *payloadPublisher) publish(ctx context.Context, signed *opsigner.SignedExecutionPayloadEnvelope) {
	block := uint64(signed.Envelope.ExecutionPayload.BlockNumber)
	if p.lastBlock != 0 && block > p.lastBlock+1 {
		p.log.Warn("signed-payload feed gap; the skipped blocks were not published",
			"from", p.lastBlock, "to", block, "missing", block-p.lastBlock-1)
	}
	p.lastBlock = block

	if err := p.out.PublishSignedL2Payload(ctx, signed); err != nil {
		p.log.Error("failed to publish signed unsafe payload to gossip",
			"block", block, "hash", signed.Envelope.ExecutionPayload.BlockHash, "err", err)
		return
	}
	p.log.Info("published signed unsafe payload to gossip",
		"block", block, "hash", signed.Envelope.ExecutionPayload.BlockHash)
}
