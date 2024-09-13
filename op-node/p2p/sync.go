package p2p

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math/big"
	"runtime/debug"
	"time"

	g "github.com/anacrolix/generics"
	"github.com/anacrolix/missinggo/v2/panicif"
	"github.com/anacrolix/sync"

	"github.com/anacrolix/chansync"
	_ "github.com/anacrolix/envpprof"
	"github.com/golang/snappy"
	"github.com/hashicorp/golang-lru/v2/simplelru"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"golang.org/x/time/rate"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

const (
	// How many payloads to allow to be cached. They will always be preferred to stack against the
	// top of the active request.
	quarantineLimit = 10
)

// StreamCtxFn provides a new context to use when handling stream requests
type StreamCtxFn func() context.Context

// Note: the mocknet in testing does not support read/write stream timeouts, the timeouts are only applied if available.
// Rate-limits always apply, and are making sure the request/response throughput is not too fast, instead of too slow.
const (
	// timeout for opening a req-resp stream to another peer. This may involve some protocol negotiation.
	streamTimeout = time.Second * 5
	// timeout for writing the request as client. Can be as long as serverReadRequestTimeout
	clientWriteRequestTimeout = time.Second * 10
	// timeout for reading a response of a serving peer as client. Can be as long as serverWriteChunkTimeout
	clientReadResponsetimeout = time.Second * 10
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

type resultCode byte

const (
	ResultCodeSuccess     resultCode = 0
	ResultCodeNotFoundErr resultCode = 1
	ResultCodeInvalidErr  resultCode = 2
	ResultCodeUnknownErr  resultCode = 3
)

var resultCodeString = []string{
	"success",
	"not found",
	"invalid request",
	"unknown error",
}

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

type newStreamFn func(ctx context.Context, peerId peer.ID, protocolId ...protocol.ID) (network.Stream, error)

type receivePayloadFn func(ctx context.Context, from peer.ID, payload *eth.ExecutionPayloadEnvelope) error

func (r receivePayloadFn) OnUnsafeL2Payload(ctx context.Context, from peer.ID, msg *eth.ExecutionPayloadEnvelope, _ PayloadSource) error {
	return r(ctx, from, msg)
}

type rangeRequest struct {
	start uint64
	end   eth.L2BlockRef
}

type syncResult struct {
	payload *eth.ExecutionPayloadEnvelope
	peer    peer.ID
}

type SyncClientMetrics interface {
	ClientPayloadByNumberEvent(num uint64, resultCode byte, duration time.Duration)
	PayloadsQuarantineSize(n int)
}

type SyncPeerScorer interface {
	onValidResponse(id peer.ID)
	onResponseError(id peer.ID)
	onRejectedPayload(id peer.ID)
}

// SyncClient implements a reverse chain sync with a minimal interface:
// signal the desired range, and receive blocks within this range back.
// Through parent-hash verification, received blocks are all ensured to be part of the canonical chain at one point,
// but it is up to the user to organize and process the results further.
//
// For the sync-client to retrieve any data, peers must be added with AddPeer(id), and removed upon disconnect with RemovePeer(id).
// The client is started with Start(), and may be started before or after changing any peers.
//
// ### Stages
//
// The sync mechanism is implemented as following:
// - User sends range request: blocks on sync main loop (with ctx timeout)
// - Main loop processes range request (from high to low), dividing block requests by number between parallel peers.
//   - The high part of the range has a known block-hash, and is marked as trusted.
//   - Once there are no more peers available for buffering requests, we stop the range request processing.
//   - Every request buffered for a peer is tracked as in-flight, by block number.
//   - In-flight requests are not repeated
//   - Requests for data that's already in the quarantine are not repeated
//   - Data already in the quarantine that is trusted is attempted to be promoted.
//
// - Peers each has their own routine for processing requests.
//   - They fetch the requested block by number, parse and validate it, and then send it back to the main loop
//   - If peers fail to fetch or process it, or fail to send it back to the main loop within timeout,
//     then doRequest returns an error. It then marks the in-flight request as completed.
//
// - Main loop receives results synchronously with the range requests
//   - The result is removed from in-flight tracker
//   - The result is added to the quarantine
//   - If we trust the hash, we try to promote the result.
//
// ### Concepts
//
// The main concepts are:
// - Quarantine: an LRU that stores the latest fetched block data, by hash as well as an extra index by number.
//
//   - Quarantine eviction: upon regular LRU eviction, or explicit removal (when we learn data is not canonical),
//     the sync result is removed from quarantine without being forwarded to the receiver.
//     The peer that provided the data may be down-scored for providing un-utilized data if the data
//     is not trusted during eviction.
//
// - Trusted data: data becomes trusted through 2 ways:
//   - The hash / parent-hash of the sync target is marked as trusted.
//   - The parent-hash of any promoted data is marked as trusted.
//
// - The trusted-data is maintained in LRU: we only care about the recent accessed blocks.
//
//   - Result promotion: content from the quarantine is "promoted" when we find the blockhash is trusted.
//     The data is removed from the quarantine, and forwarded to the receiver.
//
// ### Usage
//
// The user is expected to request the range of blocks between its existing chain head,
// and a trusted future block-hash as reference to sync towards.
// Upon receiving results from the sync-client, the user should adjust down its sync-target
// based on the received results, to avoid duplicating work when req-requesting an updated range.
// Range requests should still be repeated eventually however, as the sync client will give up on syncing a large range
// when it's too busy syncing.
//
// The rationale for this approach is that this sync mechanism is primarily intended
// for quickly filling gaps between an existing chain and a gossip chain, and not for very long block ranges.
// Syncing in the execution-layer (through snap-sync) is more appropriate for long ranges.
// If the user does sync a long range of blocks through this mechanism,
// it does end up traversing through the chain, but receives the blocks in reverse order.
// It is up to the user to persist the blocks for later processing, or drop & resync them if persistence is limited.
// TODO: Should this be renamed to AltSyncClient?
type SyncClient struct {
	log log.Logger

	cfg *rollup.Config

	metrics   SyncClientMetrics
	appScorer SyncPeerScorer

	newStreamFn        newStreamFn
	payloadByNumber    protocol.ID
	NewPeerRateLimiter func() *rate.Limiter

	peersLock sync.Mutex
	// syncing worker per peer
	peers map[peer.ID]context.CancelFunc

	syncClientRequestState

	receivePayload L2PayloadIn

	// Global rate limiter for all peers.
	globalRL *rate.Limiter

	// resource context: all peers and mainLoop tasks inherit this, and start shutting down once resCancel() is called.
	resCtx    context.Context
	resCancel context.CancelFunc

	// wait group: wait for the resources to close. Adding to this is only safe if the peersLock is held.
	wg sync.WaitGroup

	// Don't allow anything to be added to the wait-group while, or after, we are shutting down.
	// This is protected by peersLock.
	closingPeers bool

	extra               ExtraHostFeatures
	syncOnlyReqToStatic bool
}

type syncClientRequestState struct {
	mu          sync.Mutex
	nextPromote g.Option[nextPromote]
	// These are *inclusive*.
	endBlockNumber    blockNumber
	startBlockNumber  blockNumber
	wanted            map[blockNumber]*wantedBlock
	requestBlocksCond chansync.BroadcastCond
	promoterCond      chansync.BroadcastCond
}

type nextPromote struct {
	num  blockNumber
	hash common.Hash
}

func NewSyncClient(
	log log.Logger,
	cfg *rollup.Config,
	host HostNewStream,
	rcv L2PayloadIn,
	metrics SyncClientMetrics,
	appScorer SyncPeerScorer,
) *SyncClient {
	ctx, cancel := context.WithCancel(context.Background())

	c := &SyncClient{
		log:       log,
		cfg:       cfg,
		metrics:   metrics,
		appScorer: appScorer,

		newStreamFn:     host.NewStream,
		payloadByNumber: PayloadByNumberProtocolID(cfg.L2ChainID),
		// Implement the same rate limits as the server does per-peer,
		// so we don't be too aggressive to the server.
		NewPeerRateLimiter: func() *rate.Limiter {
			return rate.NewLimiter(peerServerBlocksRateLimit, peerServerBlocksBurst)
		},

		peers:          make(map[peer.ID]context.CancelFunc),
		globalRL:       rate.NewLimiter(globalServerBlocksRateLimit, globalServerBlocksBurst),
		resCtx:         ctx,
		resCancel:      cancel,
		receivePayload: rcv,
	}
	if extra, ok := host.(ExtraHostFeatures); ok && extra.SyncOnlyReqToStatic() {
		c.extra = extra
		c.syncOnlyReqToStatic = true
	}

	return c
}

func (s *SyncClient) Start() {
	s.peersLock.Lock()
	s.wg.Add(1)
	s.peersLock.Unlock()
	go s.promoter(s.resCtx)
}

func (s *SyncClient) AddPeer(id peer.ID) {
	s.peersLock.Lock()
	defer s.peersLock.Unlock()
	if s.closingPeers {
		return
	}
	if _, ok := s.peers[id]; ok {
		s.log.Debug("cannot register peer for sync duties, peer was already registered", "peer", id)
		return
	}
	s.wg.Add(1)
	// add new peer routine
	ctx, cancel := context.WithCancel(s.resCtx)
	s.peers[id] = cancel
	go s.peerLoop(ctx, id)
}

func (s *SyncClient) RemovePeer(id peer.ID) {
	s.peersLock.Lock()
	defer s.peersLock.Unlock()
	cancel, ok := s.peers[id]
	if !ok {
		s.log.Debug("cannot remove peer from sync duties, peer was not registered", "peer", id)
		return
	}
	cancel() // once loop exits
	delete(s.peers, id)
}

// Close will shut down the sync client and all attached work, and block until shutdown is complete.
// This will block if the Start() has not created the main background loop.
func (s *SyncClient) Close() error {
	s.peersLock.Lock()
	s.closingPeers = true
	s.peersLock.Unlock()
	s.resCancel()
	s.wg.Wait()
	return nil
}

func (s *SyncClient) RequestL2Range(ctx context.Context, start, end eth.L2BlockRef) error {
	if end == (eth.L2BlockRef{}) {
		s.log.Debug("P2P sync client received range signal, but cannot sync open-ended chain: need sync target to verify blocks through parent-hashes", "start", start)
		return nil
	}
	s.onRangeRequest(ctx, rangeRequest{start: start.Number, end: end})
	return nil
}

// This just emulates old behaviour for a test. It's probably pointless.
func (s *syncClientRequestState) isInFlight(ctx context.Context, num blockNumber) (inFlight bool, _ error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	wanted := s.getWantedBlock(num)
	if wanted == nil {
		return
	}
	inFlight = len(wanted.quarantined) != 0 || wanted.done.IsSet()
	return
}

// go-ethereum should expose a method for this.
type blockNumber = uint64

func (s *syncClientRequestState) trimOutsideWanted() {
	for num, wanted := range s.wanted {
		if num < s.startBlockNumber || num > s.endBlockNumber {
			wanted.done.Set()
			delete(s.wanted, num)
		}
	}
}

func (s *syncClientRequestState) addMissingWanted(log log.Logger) {
	g.MakeMapIfNilWithCap(&s.wanted, s.endBlockNumber-s.startBlockNumber+1)
	for num := s.endBlockNumber; num >= s.startBlockNumber; num-- {
		log.Debug("Scheduling P2P block request", "num", num)
		blockState, ok := s.wanted[num]
		if ok {
			blockState.promoted = false
			blockState.done.Clear()
			blockState.finalHash.SetNone()
			continue
		}
		wanted := &wantedBlock{
			num: num,
			// Set to the smallest amount that still ensures no single peer blocks us. Old behaviour was essentially 1.
			requestConcurrency: chansync.NewSemaphore(2),
		}
		s.wanted[num] = wanted
	}
}

// onRangeRequest is exclusively called by the main loop, and has thus direct access to the request bookkeeping state.
// This function transforms requested block ranges into work for each peer.
func (s *SyncClient) onRangeRequest(ctx context.Context, req rangeRequest) {
	log := s.log.New("target", req.start, "end", req.end)
	log.Info("processing L2 range request")

	s.mu.Lock()
	defer s.mu.Unlock()
	s.endBlockNumber = req.end.Number - 1
	s.startBlockNumber = req.start + 1
	s.trimOutsideWanted()
	s.addMissingWanted(log)
	s.setNextPromote(s.endBlockNumber, req.end.ParentHash)
	s.requestBlocksCond.Broadcast()
	s.promoterCond.Broadcast()
}

func (s *SyncClient) removeFromQuarantine(bn blockNumber, hash common.Hash) {
	wanted := s.getWantedBlock(bn)
	g.MustDelete(wanted.quarantined, hash)
	s.requestBlocksCond.Broadcast()
}

func (s *SyncClient) setNextPromote(bn blockNumber, hash common.Hash) {
	if bn < s.startBlockNumber {
		// We're finished syncing!
		s.nextPromote.SetNone()
		return
	}
	s.nextPromote.Set(nextPromote{
		num:  bn,
		hash: hash,
	})
	s.deleteBadQuarantines(bn, hash)
}

func (s *SyncClient) promotedBlock(payload *eth.ExecutionPayloadEnvelope) {
	// Should we check what the previous nextPromote state was?
	bn := blockNumber(payload.ExecutionPayload.BlockNumber)
	wanted := s.getWantedBlock(bn)
	if len(wanted.quarantined) != 0 {
		// Quarantine count lowered.
		s.requestBlocksCond.Broadcast()
	}
	clear(wanted.quarantined)
	s.requestBlocksCond.Broadcast()
	panicif.True(wanted.finalHash.Set(payload.ExecutionPayload.BlockHash).Ok)
	wanted.done.Set()
	wanted.promoted = true
	// Should we pass through the just promoted block number so it can avoid wrap around instead?
	panicif.Eq(0, bn)
	s.setNextPromote(bn-1, payload.ExecutionPayload.ParentHash)
}

func (s *SyncClient) deleteBadQuarantines(bn blockNumber, expected common.Hash) {
	wanted := s.getWantedBlock(bn)
	for hash, syncRes := range wanted.quarantined {
		if hash != expected {
			s.log.Debug("evicting untrusted payload from quarantine",
				"id", syncRes.payload.ExecutionPayload.ID(),
				"peer", syncRes.peer)
			// Down-score peer for having provided us a bad block that never turned out to be canonical
			s.appScorer.onRejectedPayload(syncRes.peer)
			delete(wanted.quarantined, hash)
			s.requestBlocksCond.Broadcast()
		}
	}
}

func (s *SyncClient) tryPromote(ctx context.Context, res syncResult) {
	s.log.Debug("promoting p2p sync result", "payload", res.payload.ExecutionPayload.ID(), "peer", res.peer)

	blockNumber := blockNumber(res.payload.ExecutionPayload.BlockNumber)
	// Does this actually return an error if the block is bad? The code doesn't suggest so.
	// Fortunately the driver will timeout and reset the L2 request range but that's not ideal. Also
	// it seems this is handled by posting a message, which means we don't need to unlock to make
	// the call?
	s.mu.Unlock()
	err := s.receivePayload.OnUnsafeL2Payload(ctx, res.peer, res.payload, PayloadSourceAltSync)
	s.mu.Lock()
	// Should we log err here first?
	if s.getWantedBlock(blockNumber) == nil {
		// The range was altered to not include this block number while we were sending the payload.
		return
	}
	if err != nil {
		s.log.Warn("failed to promote payload, receiver error", "err", err)
		s.removeFromQuarantine(blockNumber, res.payload.ExecutionPayload.BlockHash)
	}
	stillNextPromote := s.nextPromote.Ok && s.nextPromote.Value.hash == res.payload.ExecutionPayload.BlockHash && s.nextPromote.Value.num == blockNumber
	// If we still want the block number, but no longer want to promote this next then we leave it
	// in quarantine so can resubmit it later if it becomes trusted again. When we promote its
	// child, we will know if this block is valid immediately and can promote it again or evict it.
	if stillNextPromote {
		s.promotedBlock(res.payload)
	}
}

func (s *SyncClient) onResultUnlocked(res syncResult) {
	payload := res.payload.ExecutionPayload
	s.log.Debug("processing p2p sync result", "payload", payload.ID(), "peer", res.peer)
	blockNum := blockNumber(payload.BlockNumber)
	// Always put it in quarantine first. If promotion fails because the receiver is too busy, this functions as cache.
	s.mu.Lock()
	// panics suck
	defer s.mu.Unlock()
	wanted := s.getWantedBlock(blockNum)
	if wanted == nil {
		// We've moved on.
		return
	}
	if wanted.done.IsSet() {
		// Late arrival and we know what we should have received.
		if wanted.finalHash.Ok {
			if res.payload.ExecutionPayload.BlockHash != wanted.finalHash.Value {
				s.appScorer.onRejectedPayload(res.peer)
			}
			// We could score the peer for sending us a block here, but maybe they stalled on
			// purpose. Let's just reward the one that answered first.
		}
		return
	}
	if g.MapContains(wanted.quarantined, payload.BlockHash) {
		// We already have this block, this peer was just slow. Don't score them just as we do
		// elsewhere.
		return
	}
	if !s.trimQuarantinedBelow(blockNum) {
		// Can't fit this block there are higher priority blocks.
		return
	}
	g.MakeMapIfNil(&wanted.quarantined)
	wanted.quarantined[payload.BlockHash] = res
	s.doPayloadsQuarantineSizeMetric()
	s.promoterCond.Broadcast()
}

func (s *SyncClient) trimQuarantinedBelow(below blockNumber) bool {
	count := 0
	for bn := below; bn <= s.endBlockNumber; bn++ {
		count += len(s.getWantedBlock(bn).quarantined)
	}
	return s.trimQuarantineCache(below, count-quarantineLimit+1)
}

func (s *SyncClient) doPayloadsQuarantineSizeMetric() {
	size := 0
	for _, wanted := range s.wanted {
		size += len(wanted.quarantined)
	}
	//fmt.Printf("payloads quarantine size: %v\n", size)
	s.metrics.PayloadsQuarantineSize(size)
}

// This enforces sequential handling of promoting blocks to the driver. I assume this was a desired
// feature, it's probably expensive to run them.
func (s *SyncClient) promoter(ctx context.Context) {
	defer s.wg.Done()
	for {
		s.mu.Lock()
		s.maybePromote(ctx)
		cond := s.promoterCond.Signaled()
		s.mu.Unlock()
		select {
		case <-cond:
		case <-ctx.Done():
			return
		}
	}
}

func (s *SyncClient) maybePromote(ctx context.Context) {
	for {
		if !s.nextPromote.Ok {
			return
		}
		wanted := s.getWantedBlock(s.nextPromote.Value.num)
		if len(wanted.quarantined) == 0 {
			return
		}
		// Should have been trimmed due to knowing what the next hash should be.
		panicif.GreaterThan(len(wanted.quarantined), 1)
		for hash, syncRes := range wanted.quarantined {
			panicif.NotEq(hash, s.nextPromote.Value.hash)
			s.tryPromote(ctx, syncRes)
			// Paranoid. I know Go will shift things underneath us on maps if it can get away with
			// it.
			break
		}
	}
}

func (s *syncClientPeer) requestAndHandleResult(ctx context.Context, wanted *wantedBlock) (err error) {
	// We already established the peer is available w.r.t. rate-limiting,
	// and this is the only loop over this peer, so we can request now.

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	go func() {
		defer cancel()
		select {
		case <-wanted.done.On():
		case <-ctx.Done():
		}
	}()
	start := time.Now()

	resultCode := ResultCodeSuccess
	num := wanted.num

	envelope, err := s.doRequestRecoveringPanic(ctx, num)

	if err != nil {
		log.Warn("failed p2p sync request", "num", num, "err", err)
		resultCode = ResultCodeNotFoundErr
		sendResponseError := true

		var re requestResultErr
		if errors.As(err, &re) {
			resultCode = re.ResultCode()
			if resultCode == ResultCodeNotFoundErr {
				sendResponseError = false // don't penalize peer for this error
			}
		}

		if sendResponseError {
			s.appScorer.onResponseError(s.remoteId)
			// If we hit an error, then count it as many requests.
			// We'd like to avoid making more requests for a while, so back off.
			s.lastRequestError = time.Now()
		}
	} else {
		s.onResultUnlocked(syncResult{payload: envelope, peer: s.remoteId})
		log.Debug("completed p2p sync request", "num", num)
		s.appScorer.onValidResponse(s.remoteId)
	}

	took := time.Since(start)
	s.metrics.ClientPayloadByNumberEvent(num, byte(resultCode), took)
	return
}

func (s *syncClientPeer) requestBlocks(ctx context.Context) (err error) {
	quarantineCount := 0
	for bn := s.endBlockNumber; bn >= s.startBlockNumber; bn-- {
		wanted := s.getWantedBlock(bn)
		if wanted == nil {
			// Request range has been altered.
			break
		}
		// Don't bother to quarantine more than one block for each number.
		if len(wanted.quarantined) != 0 {
			quarantineCount += len(wanted.quarantined)
			continue
		}
		if wanted.done.IsSet() {
			continue
		}
		if quarantineCount >= quarantineLimit {
			if !s.trimQuarantineCache(bn, quarantineCount-quarantineLimit+1) {
				return
			}
		}
		// Our quarantine count could be invalid.
		return s.reserveAndRequest(ctx, wanted)
	}
	return
}

// Waits for reservations and then does the request if appropriate.
func (s *syncClientPeer) reserveAndRequest(ctx context.Context, wanted *wantedBlock) (err error) {
	s.mu.Unlock()
	defer s.mu.Lock()
	r := s.rl.Reserve()
	if !r.OK() {
		panic(s.rl)
	}
	// Undo the reservation if we never get to request. We do the peer reservation first so we don't
	// tie up other peers.
	requested := false
	defer func() {
		if !requested {
			r.Cancel()
		}
	}()
	// Wait for the peer reservation to be ready.
	select {
	case <-ctx.Done():
		return context.Cause(ctx)
	case <-wanted.done.On():
		return
	case <-time.After(r.Delay()):
	}
	select {
	default:
		// Too many on this block.
		return
	case wanted.requestConcurrency.Acquire() <- struct{}{}:
	}
	// Releasing our slot on the block could free up another peer.
	defer s.requestBlocksCond.Broadcast()
	defer func() {
		<-wanted.requestConcurrency.Release()
	}()
	err = s.globalRL.Wait(ctx)
	if err != nil {
		err = fmt.Errorf("waiting for global rate limiter: %w", err)
		return
	}
	err = s.requestAndHandleResult(ctx, wanted)
	requested = true
	if err != nil {
		err = fmt.Errorf("requesting block %v: %w", wanted.num, err)
	}
	return
}

// peerLoop for syncing from a single peer
func (s *SyncClient) peerLoop(ctx context.Context, id peer.ID) {
	defer func() {
		s.peersLock.Lock()
		delete(s.peers, id) // clean up
		s.log.Debug("stopped syncing loop of peer", "id", id)
		s.wg.Done()
		s.peersLock.Unlock()
	}()

	peer := &syncClientPeer{
		SyncClient: s,
		rl:         s.NewPeerRateLimiter(),
		remoteId:   id,
	}
	peer.Run(ctx)
}

func (s *syncClientPeer) Run(ctx context.Context) {
	log := s.log.New("peer", s.remoteId)
	log.Info("Starting P2P sync client event loop")
	for {
		s.mu.Lock()
		// if onlyReqToStatic is on, ensure that only static peers are dealing with the request.
		// Take this before requesting blocks, so if we return for whatever reason we can jump
		// straight back in if our preconditions changed.
		requestBlocksSignal := s.requestBlocksCond.Signaled()
		if s.syncOnlyReqToStatic && !s.extra.IsStatic(s.remoteId) {
			// for non-static peers, set requestBlocksCond to nil
			// this will effectively make the peer loop not perform outgoing sync-requests.
			// while sync-requests will block, the loop may still process other events (if added in the future).
			requestBlocksSignal = nil
		} else {
			err := s.requestBlocks(ctx)
			if err != nil {
				log.Warn("error requesting blocks", "err", err)
				naughtyBoyChan := make(chan struct{})
				requestBlocksSignal = naughtyBoyChan
				// Calculated approximately from the penalty that was applied to the peer rate
				// limiter in the code that used to artificially apply tokens to achieve backoff.
				time.AfterFunc(5*time.Second, func() { close(naughtyBoyChan) })
			}
		}
		s.mu.Unlock()
		// once the peer is available, wait for a sync request.
		select {
		case <-requestBlocksSignal:
		case <-ctx.Done():
			return
		}
	}

}

type requestResultErr resultCode

func (r requestResultErr) Error() string {
	var errStr string
	if ri := int(r); ri < len(resultCodeString) {
		errStr = resultCodeString[ri]
	} else {
		errStr = "invalid code"
	}
	return fmt.Sprintf("peer failed to serve request with code %d: %s", uint8(r), errStr)
}

func (r requestResultErr) ResultCode() resultCode {
	return resultCode(r)
}

func (s *syncClientPeer) doRequestRecoveringPanic(
	ctx context.Context, expectedBlockNum uint64,
) (
	envelope *eth.ExecutionPayloadEnvelope, err error,
) {
	err = panicGuard(s.log, fmt.Sprintf("doing alt sync request to %v", s.remoteId), func() (err error) {
		envelope, err = s.doRequest(ctx, expectedBlockNum)
		return
	})
	return
}

func (s *syncClientPeer) doRequest(
	ctx context.Context, expectedBlockNum uint64,
) (
	envelope *eth.ExecutionPayloadEnvelope, err error,
) {
	// open stream to peer
	reqCtx, reqCancel := context.WithTimeout(ctx, streamTimeout)
	str, err := s.newStreamFn(reqCtx, s.remoteId, s.payloadByNumber)
	reqCancel()
	if err != nil {
		err = fmt.Errorf("failed to open stream: %w", err)
		return
	}
	defer str.Close()
	// set write timeout (if available)
	_ = str.SetWriteDeadline(time.Now().Add(clientWriteRequestTimeout))
	err = binary.Write(str, binary.LittleEndian, expectedBlockNum)
	if err != nil {
		err = fmt.Errorf("failed to write request (%d): %w", expectedBlockNum, err)
		return
	}
	err = str.CloseWrite()
	if err != nil {
		err = fmt.Errorf("failed to close writer side while making request: %w", err)
		return
	}

	// set read timeout (if available)
	_ = str.SetReadDeadline(time.Now().Add(clientReadResponsetimeout))

	// Limit input, as well as output.
	// Compression may otherwise continue to read ignored data for a small output,
	// or output more data than desired (zip-bomb)
	r := io.LimitReader(str, maxGossipSize)
	var result [1]byte
	_, err = io.ReadFull(r, result[:])
	if err != nil {
		err = fmt.Errorf("failed to read result code: %w", err)
		return
	}
	if res := resultCode(result[0]); res != ResultCodeSuccess {
		err = requestResultErr(res)
		return
	}
	var versionData [4]byte
	_, err = io.ReadFull(r, versionData[:])
	if err != nil {
		err = fmt.Errorf("failed to read version part of response: %w", err)
		return
	}

	// payload is SSZ encoded with Snappy framed compression
	// snappy sux, gross
	r = snappy.NewReader(r)
	r = io.LimitReader(r, maxGossipSize)

	// We cannot stream straight into the SSZ decoder, since we need the scope of the SSZ payload.
	// The server does not prepend it, nor would we trust a claimed length anyway, so we buffer the data we get.
	data, err := io.ReadAll(r)
	if err != nil {
		err = fmt.Errorf("failed to read response: %w", err)
		return
	}

	version := binary.LittleEndian.Uint32(versionData[:])
	isCanyon := s.cfg.IsCanyon(s.cfg.TimestampForBlock(expectedBlockNum))
	envelope, err = readExecutionPayload(version, data, isCanyon)
	if err != nil {
		err = fmt.Errorf("reading execution payload: %w", err)
		return
	}
	err = str.CloseRead()
	if err != nil {
		err = fmt.Errorf("failed to close reading side: %w", err)
		return
	}
	err = verifyBlock(envelope, expectedBlockNum)
	if err != nil {
		err = fmt.Errorf("received execution payload is invalid: %w", err)
		return
	}
	return
}

// Modelled on "net/http".conn.serve's panic handler. Retained separately for testing purposes. Logs
// the panic details, and returns the function error. Possibly f should not return an error.
func panicGuard(logger log.Logger, msg string, f func() error, logAttrs ...any) (err error) {
	defer func() {
		r := recover()
		if r == nil {
			return
		}
		// I'd make this crit but clearly someone sees it fairly regularly and it is libp2p...
		logger.Error(
			fmt.Sprintf("panic %s: %v\n%s", msg, r, debug.Stack()),
			logAttrs...)
		// This shouldn't be possible, since f can't assign an error if it panicked.
		if err != nil {
			logger.Error(fmt.Sprintf("unhandled error %s: %v", msg, err))
		}
		err = fmt.Errorf("panic: %v", r)
	}()
	err = f()
	return
}

// readExecutionPayload will unmarshal the supplied data into an ExecutionPayloadEnvelope.
func readExecutionPayload(version uint32, data []byte, isCanyon bool) (*eth.ExecutionPayloadEnvelope, error) {
	switch version {
	case 0:
		blockVersion := eth.BlockV1
		if isCanyon {
			blockVersion = eth.BlockV2
		}
		var res eth.ExecutionPayload
		if err := res.UnmarshalSSZ(blockVersion, uint32(len(data)), bytes.NewReader(data)); err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}
		return &eth.ExecutionPayloadEnvelope{ExecutionPayload: &res}, nil
	case 1:
		envelope := &eth.ExecutionPayloadEnvelope{}
		if err := envelope.UnmarshalSSZ(uint32(len(data)), bytes.NewReader(data)); err != nil {
			return nil, fmt.Errorf("failed to decode execution payload envelope response: %w", err)
		}
		return envelope, nil
	default:
		return nil, fmt.Errorf("unrecognized version: %d", version)
	}
}

func verifyBlock(envelope *eth.ExecutionPayloadEnvelope, expectedNum uint64) error {
	payload := envelope.ExecutionPayload

	// verify L2 block
	if expectedNum != uint64(payload.BlockNumber) {
		return fmt.Errorf("received execution payload for block %d, but expected block %d", payload.BlockNumber, expectedNum)
	}
	actual, ok := envelope.CheckBlockHash()
	if !ok { // payload itself contains bad block hash
		return fmt.Errorf("received execution payload for block %d with bad block hash %s, expected %s", expectedNum, payload.BlockHash, actual)
	}
	return nil
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

	GlobalRequestsRL *rate.Limiter
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
		GlobalRequestsRL: globalRequestsRL,
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

	// Doesn't look like rate limiting gets special treatment.
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
		_, _ = stream.Write([]byte{byte(resultCode)})
	} else {
		log.Debug("successfully served sync response", "req", req)
	}
	srv.metrics.ServerPayloadByNumberEvent(req, byte(resultCode), time.Since(start))
}

var errInvalidRequest = errors.New("invalid request")

func (srv *ReqRespServer) handleSyncRequest(ctx context.Context, stream network.Stream) (uint64, error) {
	peerId := stream.Conn().RemotePeer()

	// find rate limiting data of peer, or add otherwise
	srv.peerStatsLock.Lock()
	ps, _ := srv.peerRateLimits.Get(peerId)
	if ps == nil {
		ps = &peerStat{
			Requests: rate.NewLimiter(rate.Inf, 0),
		}
		srv.peerRateLimits.Add(peerId, ps)
		ps.Requests.Reserve() // count the hit, but make it delay the next request rather than immediately waiting
	} else {
		// Only wait if it's an existing peer, otherwise the instant rate-limit Wait call always errors.

		// If the requester thinks we're taking too long, then it's their problem and they can disconnect.
		// We'll disconnect ourselves only when failing to read/write,
		// if the work is invalid (range validation), or when individual sub tasks timeout.
		if err := ps.Requests.Wait(ctx); err != nil {
			return 0, fmt.Errorf("timed out waiting for global sync rate limit: %w", err)
		}
	}
	srv.peerStatsLock.Unlock()

	// Take the global rate limiter after the peer-specific one so as not to waste tokens on peers that can't advance.
	// take a token from the global rate-limiter,
	// to make sure there's not too much concurrent server work between different peers.
	if err := srv.GlobalRequestsRL.Wait(ctx); err != nil {
		return 0, fmt.Errorf("timed out waiting for global sync rate limit: %w", err)
	}

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

// State for blocks that are or have been requested through the sync client.
type wantedBlock struct {
	num                blockNumber
	requestConcurrency chansync.Semaphore
	// On when we've either promoted the block, or no longer want it.
	done chansync.Flag
	// Whether we're done because we submitted a block upstream. If we get a request for a range
	// that includes a block that's been promoted, we should get it again.
	promoted bool
	// This prevents duplicate requests when one delivers.
	quarantined map[common.Hash]syncResult
	// The correct hash when it becomes known. Can be used to score late replies.
	finalHash g.Option[common.Hash]
}

type syncClientPeer struct {
	*SyncClient
	lastRequestError time.Time
	rl               *rate.Limiter
	remoteId         peer.ID
}

// Returns the request state. Can be nil if the request range has been altered.
func (s *syncClientRequestState) getWantedBlock(blockNum blockNumber) *wantedBlock {
	return s.wanted[blockNum]
}

// Returns true if all the space required was trimmed from quarantined payloads before the given
// block number.
func (s *SyncClient) trimQuarantineCache(before blockNumber, spaceRequired int) bool {
	for num := s.startBlockNumber; spaceRequired > 0 && num < before; num++ {
		wanted := s.getWantedBlock(num)
		for hash := range wanted.quarantined {
			delete(wanted.quarantined, hash)
			s.requestBlocksCond.Broadcast()
			spaceRequired--
		}
	}
	return spaceRequired <= 0
}
