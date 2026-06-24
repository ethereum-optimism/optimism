package p2p

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/golang/snappy"
	"github.com/hashicorp/golang-lru/v2/simplelru"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"golang.org/x/time/rate"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// StreamCtxFn provides a new context to use when handling stream requests
type StreamCtxFn func() context.Context

// Note: the mocknet in testing does not support read/write stream timeouts, the timeouts are only applied if available.
// Rate-limits always apply, and are making sure the request/response throughput is not too fast, instead of too slow.
const (
	// timeout for reading the request content, deny the request if it cannot be fully read in time
	serverReadRequestTimeout = time.Second * 10
	// timeout for writing a single response message chunk
	// (if a future response consists of multiple chunks, reset the writing timeout per chunk)
	serverWriteChunkTimeout = time.Second * 10
	// after the rate-limit reservation hits the max throttle delay, give up on serving a request and just close the stream
	maxThrottleDelay = time.Second * 20
	// Do not serve more than 20 requests per second
	globalServerBlocksRateLimit rate.Limit = 20
	// Allows a burst of 2x our rate limit
	globalServerBlocksBurst = 40
	// Do not serve more than 4 requests per second to the same peer, so we can serve other peers at the same time
	peerServerBlocksRateLimit rate.Limit = 4
	// Allow a peer to request 30s of blocks at once
	peerServerBlocksBurst = 15
)

const (
	ResultCodeSuccess     byte = 0
	ResultCodeNotFoundErr byte = 1
	ResultCodeInvalidErr  byte = 2
	ResultCodeUnknownErr  byte = 3
)

func PayloadByNumberProtocolID(l2ChainID *big.Int) protocol.ID {
	return protocol.ID(fmt.Sprintf("/opstack/req/payload_by_number/%d/0", l2ChainID))
}

type requestHandlerFn func(ctx context.Context, log log.Logger, stream network.Stream)

func MakeStreamHandler(resourcesCtx context.Context, log log.Logger, fn requestHandlerFn) network.StreamHandler {
	return func(stream network.Stream) {
		log := log.New("peer", stream.Conn().ID(), "remote", stream.Conn().RemoteMultiaddr())
		defer func() {
			if err := recover(); err != nil {
				log.Error("p2p server request handling panic", "err", err, "protocol", stream.Protocol())
			}
		}()
		defer stream.Close()
		fn(resourcesCtx, log, stream)
	}
}

// peerStat maintains rate-limiting data of a peer that requests blocks from us.
type peerStat struct {
	// Requests tokenizes each request to sync
	Requests *rate.Limiter
}

type L2Chain interface {
	PayloadByNumber(ctx context.Context, number uint64) (*eth.ExecutionPayloadEnvelope, error)
}

type ReqRespServerMetrics interface {
	ServerPayloadByNumberEvent(num uint64, resultCode byte, duration time.Duration)
}

type ReqRespServer struct {
	cfg *rollup.Config

	l2 L2Chain

	metrics ReqRespServerMetrics

	peerRateLimits *simplelru.LRU[peer.ID, *peerStat]
	peerStatsLock  sync.Mutex

	globalRequestsRL *rate.Limiter
}

func NewReqRespServer(cfg *rollup.Config, l2 L2Chain, metrics ReqRespServerMetrics) *ReqRespServer {
	// We should never allow over 1000 different peers to churn through quickly,
	// so it's fine to prune rate-limit details past this.

	peerRateLimits, _ := simplelru.NewLRU[peer.ID, *peerStat](1000, nil)
	globalRequestsRL := rate.NewLimiter(globalServerBlocksRateLimit, globalServerBlocksBurst)

	return &ReqRespServer{
		cfg:              cfg,
		l2:               l2,
		metrics:          metrics,
		peerRateLimits:   peerRateLimits,
		globalRequestsRL: globalRequestsRL,
	}
}

// HandleSyncRequest is a stream handler function to register the L2 unsafe payloads alt-sync protocol.
// See MakeStreamHandler to transform this into a LibP2P handler function.
//
// Note that the same peer may open parallel streams.
//
// The caller must Close the stream.
func (srv *ReqRespServer) HandleSyncRequest(ctx context.Context, log log.Logger, stream network.Stream) {
	// may stay 0 if we fail to decode the request
	start := time.Now()

	// We wait as long as necessary; we throttle the peer instead of disconnecting,
	// unless the delay reaches a threshold that is unreasonable to wait for.
	ctx, cancel := context.WithTimeout(ctx, maxThrottleDelay)
	req, err := srv.handleSyncRequest(ctx, stream)
	cancel()

	resultCode := ResultCodeSuccess
	if err != nil {
		log.Warn("failed to serve p2p sync request", "req", req, "err", err)
		if errors.Is(err, ethereum.NotFound) {
			resultCode = ResultCodeNotFoundErr
		} else if errors.Is(err, errInvalidRequest) {
			resultCode = ResultCodeInvalidErr
		} else {
			resultCode = ResultCodeUnknownErr
		}
		// try to write error code, so the other peer can understand the reason for failure.
		_, _ = stream.Write([]byte{resultCode})
	} else {
		log.Debug("successfully served sync response", "req", req)
	}
	srv.metrics.ServerPayloadByNumberEvent(req, resultCode, time.Since(start))
}

var errInvalidRequest = errors.New("invalid request")

func (srv *ReqRespServer) handleSyncRequest(ctx context.Context, stream network.Stream) (uint64, error) {
	peerId := stream.Conn().RemotePeer()

	// take a token from the global rate-limiter,
	// to make sure there's not too much concurrent server work between different peers.
	if err := srv.globalRequestsRL.Wait(ctx); err != nil {
		return 0, fmt.Errorf("timed out waiting for global sync rate limit: %w", err)
	}

	// find rate limiting data of peer, or add otherwise
	srv.peerStatsLock.Lock()
	ps, _ := srv.peerRateLimits.Get(peerId)
	if ps == nil {
		ps = &peerStat{
			Requests: rate.NewLimiter(peerServerBlocksRateLimit, peerServerBlocksBurst),
		}
		srv.peerRateLimits.Add(peerId, ps)
		ps.Requests.Reserve() // count the hit, but make it delay the next request rather than immediately waiting
	} else {
		// Only wait if it's an existing peer, otherwise the instant rate-limit Wait call always errors.

		// If the requester thinks we're taking too long, then it's their problem and they can disconnect.
		// We'll disconnect ourselves only when failing to read/write,
		// if the work is invalid (range validation), or when individual sub tasks timeout.
		if err := ps.Requests.Wait(ctx); err != nil {
			srv.peerStatsLock.Unlock()
			return 0, fmt.Errorf("timed out waiting for global sync rate limit: %w", err)
		}
	}
	srv.peerStatsLock.Unlock()

	// Set read deadline, if available
	_ = stream.SetReadDeadline(time.Now().Add(serverReadRequestTimeout))

	// Read the request
	var req uint64
	if err := binary.Read(stream, binary.LittleEndian, &req); err != nil {
		return 0, fmt.Errorf("failed to read requested block number: %w", err)
	}
	if err := stream.CloseRead(); err != nil {
		return req, fmt.Errorf("failed to close reading-side of a P2P sync request call: %w", err)
	}

	// Check the request is within the expected range of blocks
	if req < srv.cfg.Genesis.L2.Number {
		return req, fmt.Errorf("cannot serve request for L2 block %d before genesis %d: %w", req, srv.cfg.Genesis.L2.Number, errInvalidRequest)
	}
	max, err := srv.cfg.TargetBlockNumber(uint64(time.Now().Unix()))
	if err != nil {
		return req, fmt.Errorf("cannot determine max target block number to verify request: %w", errInvalidRequest)
	}
	if req > max {
		return req, fmt.Errorf("cannot serve request for L2 block %d after max expected block (%v): %w", req, max, errInvalidRequest)
	}

	envelope, err := srv.l2.PayloadByNumber(ctx, req)
	if err != nil {
		if errors.Is(err, ethereum.NotFound) {
			return req, fmt.Errorf("peer requested unknown block by number: %w", err)
		} else {
			return req, fmt.Errorf("failed to retrieve payload to serve to peer: %w", err)
		}
	}

	// We set write deadline, if available, to safely write without blocking on a throttling peer connection
	_ = stream.SetWriteDeadline(time.Now().Add(serverWriteChunkTimeout))

	w := snappy.NewBufferedWriter(stream)

	if srv.cfg.IsEcotone(uint64(envelope.ExecutionPayload.Timestamp)) {
		// 0 - resultCode: success = 0
		// 1:5 - version: 1 (little endian)
		tmp := [5]byte{0, 1, 0, 0, 0}
		if _, err := stream.Write(tmp[:]); err != nil {
			return req, fmt.Errorf("failed to write response header data: %w", err)
		}
		if _, err := envelope.MarshalSSZ(w); err != nil {
			return req, fmt.Errorf("failed to write payload to sync response: %w", err)
		}
	} else {
		// 0 - resultCode: success = 0
		// 1:5 - version: 0
		var tmp [5]byte
		if _, err := stream.Write(tmp[:]); err != nil {
			return req, fmt.Errorf("failed to write response header data: %w", err)
		}
		if _, err := envelope.ExecutionPayload.MarshalSSZ(w); err != nil {
			return req, fmt.Errorf("failed to write payload to sync response: %w", err)
		}
	}

	if err := w.Close(); err != nil {
		return req, fmt.Errorf("failed to finishing writing payload to sync response: %w", err)
	}

	return req, nil
}
