package main

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// getUnsafePayloadMethod is the Direct Sync single-height pull, served on the
// node's main RPC (hot tier from the replay ring, cold tier reconstructed) —
// the sidecar's backend for serving P2P req-resp payloads-by-number.
const getUnsafePayloadMethod = "opql_getUnsafePayload"

// directSyncChain implements p2p.L2Chain over the node's Direct Sync pull
// endpoint, turning the sidecar into a full req-resp citizen: gossip peers can
// backfill payloads-by-number from this node's ring/archive. Bounded by the
// node's own pull caps; heights the node no longer retains map to NotFound.
type directSyncChain struct {
	node *nodeClient
	log  log.Logger
}

func (c *directSyncChain) PayloadByNumber(ctx context.Context, number uint64) (*eth.ExecutionPayloadEnvelope, error) {
	var msg *directSyncMsg
	err := c.node.rpc.CallContext(ctx, &msg, getUnsafePayloadMethod, hexutil.EncodeUint64(number))
	if err != nil {
		return nil, fmt.Errorf("direct-sync pull for block %d failed: %w", number, err)
	}
	if msg == nil {
		// The node does not retain this height: the req-resp server maps
		// NotFound onto ResultCodeNotFoundErr for the requesting peer.
		return nil, ethereum.NotFound
	}
	decoded, err := decodeDirectSyncPayload(msg)
	if err != nil {
		return nil, fmt.Errorf("direct-sync payload for block %d failed decode: %w", number, err)
	}
	if got := decoded.blockNumber(); got != number {
		return nil, fmt.Errorf("direct-sync pull for block %d returned block %d", number, got)
	}
	return decoded.envelope, nil
}
