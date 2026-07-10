package main

import (
	"context"
	"fmt"
	"time"

	"github.com/golang/snappy"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/peer"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
	gethrpc "github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// nodeClient is the consensus-node JSON-RPC endpoint the sidecar delegates the
// unsafe-block verdict to. ws:// gives a persistent connection; http:// also
// works since the node serves both.
type nodeClient struct {
	rpc     *gethrpc.Client
	timeout time.Duration
	log     log.Logger
}

func dialNode(ctx context.Context, url string, timeout time.Duration, logger log.Logger) (*nodeClient, error) {
	c, err := gethrpc.DialContext(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("failed to dial consensus node rpc %q: %w", url, err)
	}
	return &nodeClient{rpc: c, timeout: timeout, log: logger}, nil
}

// verifyVerdict is the response of the node's admin_verifyUnsafePayload.
type verifyVerdict struct {
	Verdict string `json:"verdict"`
	Reason  string `json:"reason"`
}

// delegatingRuntimeConfig implements both p2p.GossipRuntimeConfig and
// p2p.DelegatedBlockSignatureValidator. The signer truth lives entirely in the
// consensus node: the sidecar holds no signer state, so the GossipRuntimeConfig
// signer accessors are inert and the real decision is made in
// ValidateUnsafeBlockSignature by calling the node.
type delegatingRuntimeConfig struct {
	node *nodeClient
	log  log.Logger
}

var (
	_ interface {
		P2PSequencerAddress() common.Address
		PreviousP2PSequencerAddress() common.Address
		ConfirmCurrentSigner()
	} = (*delegatingRuntimeConfig)(nil)
)

// The sidecar never verifies signatures locally, so these are inert. They are
// only consulted by the built-in path, which the delegate replaces.
func (d *delegatingRuntimeConfig) P2PSequencerAddress() common.Address { return common.Address{} }
func (d *delegatingRuntimeConfig) PreviousP2PSequencerAddress() common.Address {
	return common.Address{}
}
func (d *delegatingRuntimeConfig) ConfirmCurrentSigner() {}

// ValidateUnsafeBlockSignature delegates the gossipsub verdict to the consensus
// node. It builds a superset request that serves both node kinds:
//
//   - op-con-node reads executionPayload + parentBeaconBlockRoot + signature +
//     payloadHash (JSON engine-envelope path), and
//   - op-con-ex-node reads version + payload (raw signature-prefixed gossip
//     bytes, decoded via op-alloy).
//
// The node recovers the signer against the address it owns, ingests on accept,
// and returns the verdict. On any transport/decode error here we return
// ValidationIgnore (never Reject) so honest peers are not penalized for our own
// inability to decide — the same rule the node applies to an unknown signer.
func (d *delegatingRuntimeConfig) ValidateUnsafeBlockSignature(
	ctx context.Context,
	chainID eth.ChainID,
	version eth.BlockVersion,
	signature eth.Bytes65,
	payloadBytes []byte,
) pubsub.ValidationResult {
	req, err := buildVerifyRequest(version, signature, payloadBytes)
	if err != nil {
		d.log.Warn("failed to build verify request; ignoring block", "err", err)
		return pubsub.ValidationIgnore
	}

	callCtx, cancel := context.WithTimeout(ctx, d.node.timeout)
	defer cancel()

	var verdict verifyVerdict
	if err := d.node.rpc.CallContext(callCtx, &verdict, "admin_verifyUnsafePayload", req); err != nil {
		// Node unreachable / errored: we cannot decide. Ignore, do not penalize.
		d.log.Warn("admin_verifyUnsafePayload call failed; ignoring block", "err", err)
		return pubsub.ValidationIgnore
	}

	switch verdict.Verdict {
	case "accept":
		return pubsub.ValidationAccept
	case "reject":
		d.log.Debug("node rejected gossiped unsafe block", "reason", verdict.Reason)
		return pubsub.ValidationReject
	case "ignore":
		d.log.Debug("node could not attribute gossiped unsafe block", "reason", verdict.Reason)
		return pubsub.ValidationIgnore
	default:
		d.log.Warn("unknown verdict from node; ignoring block", "verdict", verdict.Verdict)
		return pubsub.ValidationIgnore
	}
}

// buildVerifyRequest assembles the UNIFIED request for
// admin_verifyUnsafePayload: `{version, payload}` where payload is the
// canonical `snappy(signature(65) || SSZ)` envelope — the exact Direct Sync
// carrier bytes, one shape for both node kinds.
func buildVerifyRequest(version eth.BlockVersion, signature eth.Bytes65, payloadBytes []byte) (map[string]any, error) {
	raw := make([]byte, 0, len(signature)+len(payloadBytes))
	raw = append(raw, signature[:]...)
	raw = append(raw, payloadBytes...)
	return map[string]any{
		"version": versionToInt(version),
		"payload": hexutil.Encode(snappy.Encode(nil, raw)),
	}, nil
}

func versionToInt(v eth.BlockVersion) int {
	switch v {
	case eth.BlockV1:
		return 1
	case eth.BlockV2:
		return 2
	case eth.BlockV3:
		return 3
	case eth.BlockV4:
		return 4
	default:
		return 0
	}
}

// loggingGossipIn satisfies p2p.GossipIn. Ingestion happens inside the delegate
// (verify-and-ingest in one call), so this only observes accepted deliveries.
type loggingGossipIn struct {
	log log.Logger
}

func (g *loggingGossipIn) OnUnsafeL2Payload(_ context.Context, from peer.ID, msg *eth.ExecutionPayloadEnvelope) error {
	g.log.Debug("gossip accepted unsafe payload delivered",
		"peer", from,
		"block", uint64(msg.ExecutionPayload.BlockNumber),
		"hash", msg.ExecutionPayload.BlockHash,
	)
	return nil
}
