package helpers

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/enr"

	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// EL sync emulation for the reth backend.
//
// In the op-geth action tests, EL (snap) sync works because the sequencer's and verifier's op-geth
// nodes peer over devp2p: op-node points the verifier's engine at an unsafe head it lacks, the
// engine answers SYNCING and backfills the missing blocks from its peer, and the test waits for the
// blocks to appear. The ephemeral reth engines are out-of-process subprocesses with no networking,
// so there is no devp2p to carry that backfill.
//
// This file supplies an in-process stand-in: `AddPeers`/`Enode`/`PeerCount` are wired against a
// registry that maps a synthetic enode to the reth backend it identifies, and each peering starts a
// sync pump on both engines. The pump watches the engine's sync target (`optest_syncTarget`, the
// head a forkchoice update reported SYNCING for) and, while one is set and the peer has it, copies
// the peer's blocks from where the two chains diverge up to the target, one validated
// `optest_importBlock` at a time — reproducing what op-geth's snap sync does, without the p2p. Because the pump only runs once the
// engine has actually reported SYNCING (which op-node triggers only during genuine EL sync, never
// for a CL-mode gapped payload), it activates exactly when a real EL would snap-sync and stays idle
// otherwise.

// rethPeerRegistry resolves a synthetic *enode.Node back to the reth backend it stands for, so
// AddPeers can find the peer engine to sync from. Keyed by the random per-engine id; entries are
// removed on engine shutdown.
var rethPeerRegistry = struct {
	sync.Mutex
	byID map[enode.ID]*rethBackend
}{byID: make(map[enode.ID]*rethBackend)}

func registerRethPeer(b *rethBackend) {
	rethPeerRegistry.Lock()
	rethPeerRegistry.byID[b.id] = b
	rethPeerRegistry.Unlock()
}

func lookupRethPeer(id enode.ID) *rethBackend {
	rethPeerRegistry.Lock()
	defer rethPeerRegistry.Unlock()
	return rethPeerRegistry.byID[id]
}

func deregisterRethPeer(b *rethBackend) {
	rethPeerRegistry.Lock()
	delete(rethPeerRegistry.byID, b.id)
	rethPeerRegistry.Unlock()
}

func randomEnodeID() enode.ID {
	var id enode.ID
	if _, err := rand.Read(id[:]); err != nil {
		panic(fmt.Sprintf("read random enode id: %v", err))
	}
	return id
}

// node returns an opaque *enode.Node carrying this backend's id. Only the id is meaningful: the
// action tests pass the node to a peer's AddPeers, which resolves it back to this engine via the
// registry. It is not a real, dial-able record.
func (b *rethBackend) node() *enode.Node {
	return enode.SignNull(new(enr.Record), b.id)
}

// addPeer records a symmetric peering with `peer` and starts a sync pump on each side, so either
// engine backfills from the other. Peering is bidirectional (as devp2p peering is) so both engines
// report a non-zero PeerCount, which the EL-sync tests wait on before inserting the unsafe head.
func (b *rethBackend) addPeer(peer *rethBackend) {
	b.link(peer)
	peer.link(b)
}

// link records `peer` and starts a pump backfilling this engine from it, unless already peered or
// shut down.
func (b *rethBackend) link(peer *rethBackend) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if _, known := b.peers[peer.id]; known || b.closed {
		return
	}
	b.peers[peer.id] = peer
	b.startPumpLocked(peer)
}

func (b *rethBackend) peerCount() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.peers)
}

// syncPollInterval is how often the pump checks whether the engine has been asked to sync and, if
// so, backfills a batch of blocks. The EL-sync tests wait on the result with multi-second timeouts,
// so a short interval keeps the emulated sync responsive without busy-spinning.
const syncPollInterval = 25 * time.Millisecond

func (b *rethBackend) startPumpLocked(peer *rethBackend) {
	ctx, cancel := context.WithCancel(context.Background())
	b.pumpCancels = append(b.pumpCancels, cancel)

	b.pumpWG.Add(1)
	go func() {
		defer b.pumpWG.Done()
		ticker := time.NewTicker(syncPollInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				b.pumpOnce(ctx, peer)
			}
		}
	}()
}

// pumpOnce performs one backfill pass. If the engine has reported a sync target — the head a real
// EL would snap-sync towards — and the peer has it on its canonical chain, it imports the peer's
// blocks from the highest block both canonical chains share up to the target, one validated import
// at a time in ascending order so each block's parent is already present. A block that does not
// extend the local head, because the head is on another fork, is only recorded; the forkchoice
// update towards the target then reorgs onto it. The pass stops early once the target is reached,
// or is cleared or replaced, and a target already backfilled from this peer is skipped. It
// deliberately never moves the safe/finalized pointers (those wait for op-node's forkchoice
// updates), reproducing exactly the state op-geth reaches after backfilling under a SYNCING
// forkchoice: the chain is present, but nothing is yet marked safe.
func (b *rethBackend) pumpOnce(ctx context.Context, peer *rethBackend) {
	target, err := b.syncTarget(ctx)
	if err != nil || target == (common.Hash{}) || b.backfilled(peer, target) {
		return
	}
	targetNum, found, err := peer.canonicalNumber(ctx, target)
	if err != nil || !found {
		return
	}
	localHead, err := b.blockNumber(ctx)
	if err != nil {
		return
	}
	forkPoint, found, err := b.forkPoint(ctx, peer, min(localHead, targetNum))
	if err != nil {
		return
	}
	if !found {
		b.warnSyncProblem(peer, "EL sync found no block in common with peer")
		return
	}
	for n := forkPoint + 1; n <= targetNum; n++ {
		if current, err := b.syncTarget(ctx); err != nil || current != target {
			return
		}
		payload, err := peer.blockPayload(ctx, n)
		if err != nil || payload == nil {
			return
		}
		status, err := b.importBlock(ctx, payload)
		if err != nil {
			b.warnSyncProblem(peer, "EL sync import failed", "number", n, "err", err)
			return
		}
		if status.Status != eth.ExecutionValid {
			b.warnSyncProblem(peer, "EL sync import not valid", "number", n,
				"status", status.Status, "validationError", status.ValidationError)
			return
		}
		b.clearSyncProblem(peer)
		b.log.Info("EL sync imported block from peer", "number", n, "hash", *status.LatestValidHash)
	}
	b.markBackfilled(peer, target)
}

// forkPoint returns the highest block number at or below from at which this engine's and the
// peer's canonical chains hold the same block, or found=false if they share none.
func (b *rethBackend) forkPoint(ctx context.Context, peer *rethBackend, from uint64) (number uint64, found bool, err error) {
	for n := from; ; n-- {
		local, err := b.canonicalHash(ctx, n)
		if err != nil {
			return 0, false, err
		}
		remote, err := peer.canonicalHash(ctx, n)
		if err != nil {
			return 0, false, err
		}
		if local != (common.Hash{}) && local == remote {
			return n, true, nil
		}
		if n == 0 {
			return 0, false, nil
		}
	}
}

// backfilled reports whether a pass already imported everything up to target from peer.
func (b *rethBackend) backfilled(peer *rethBackend, target common.Hash) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.syncedTo[peer.id] == target
}

func (b *rethBackend) markBackfilled(peer *rethBackend, target common.Hash) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.syncedTo == nil {
		b.syncedTo = make(map[enode.ID]common.Hash)
	}
	b.syncedTo[peer.id] = target
}

// warnSyncProblem logs a backfill failure from `peer` at Warn, once until the next successful
// import from it: the pump retries every syncPollInterval and would otherwise repeat the warning.
func (b *rethBackend) warnSyncProblem(peer *rethBackend, msg string, ctx ...any) {
	key := fmt.Sprint(append([]any{msg}, ctx...)...)
	b.mu.Lock()
	repeated := b.syncProblems[peer.id] == key
	if b.syncProblems == nil {
		b.syncProblems = make(map[enode.ID]string)
	}
	b.syncProblems[peer.id] = key
	b.mu.Unlock()
	if !repeated {
		b.log.Warn(msg, append(ctx, "peer", peer.id)...)
	}
}

func (b *rethBackend) clearSyncProblem(peer *rethBackend) {
	b.mu.Lock()
	delete(b.syncProblems, peer.id)
	b.mu.Unlock()
}

// shutdown stops every sync pump, removes the engine from the peer registry, and closes the
// subprocess. It is safe to call more than once (t.Cleanup plus an explicit L2Engine.Close).
func (b *rethBackend) shutdown() {
	b.shutdownOnce.Do(func() {
		b.mu.Lock()
		b.closed = true
		cancels := b.pumpCancels
		b.pumpCancels = nil
		b.mu.Unlock()
		for _, cancel := range cancels {
			cancel()
		}
		b.pumpWG.Wait()
		deregisterRethPeer(b)
		b.proc.Close()
	})
}

// --- sync-pump RPC helpers ---

// syncTarget reads the head the engine last reported SYNCING for and has not since resolved, or the
// zero hash when it is not behind (optest_syncTarget).
func (b *rethBackend) syncTarget(ctx context.Context) (common.Hash, error) {
	var target common.Hash
	err := b.client.CallContext(ctx, &target, "optest_syncTarget")
	return target, err
}

// canonicalNumber returns the number of block `hash` if it is on this engine's canonical chain.
func (b *rethBackend) canonicalNumber(ctx context.Context, hash common.Hash) (uint64, bool, error) {
	var byHash *rpcBlockID
	if err := b.client.CallContext(ctx, &byHash, "eth_getBlockByHash", hash, false); err != nil {
		return 0, false, err
	}
	if byHash == nil {
		return 0, false, nil
	}
	var byNumber *rpcBlockID
	if err := b.client.CallContext(ctx, &byNumber, "eth_getBlockByNumber", byHash.Number, false); err != nil {
		return 0, false, err
	}
	if byNumber == nil || byNumber.Hash != hash {
		return 0, false, nil
	}
	return uint64(byHash.Number), true, nil
}

// rpcBlockID is the part of an eth_getBlockBy* result the sync pump reads.
type rpcBlockID struct {
	Number hexutil.Uint64 `json:"number"`
	Hash   common.Hash    `json:"hash"`
}

// canonicalHash returns the hash of this engine's canonical block at number, or the zero hash if it
// has none.
func (b *rethBackend) canonicalHash(ctx context.Context, number uint64) (common.Hash, error) {
	var block *rpcBlockID
	if err := b.client.CallContext(ctx, &block, "eth_getBlockByNumber", hexutil.Uint64(number), false); err != nil {
		return common.Hash{}, err
	}
	if block == nil {
		return common.Hash{}, nil
	}
	return block.Hash, nil
}

func (b *rethBackend) blockNumber(ctx context.Context) (uint64, error) {
	var n hexutil.Uint64
	err := b.client.CallContext(ctx, &n, "eth_blockNumber")
	return uint64(n), err
}

// blockPayload fetches the canonical block at `number` from this engine as the full execution data
// an engine_newPayload would carry (optest_blockPayloadByNumber), or nil if the engine lacks it.
func (b *rethBackend) blockPayload(ctx context.Context, number uint64) (json.RawMessage, error) {
	var payload json.RawMessage
	if err := b.client.CallContext(ctx, &payload, "optest_blockPayloadByNumber", number); err != nil {
		return nil, err
	}
	if len(payload) == 0 || string(payload) == "null" {
		return nil, nil
	}
	return payload, nil
}

// importBlock validates, executes, and records a block obtained from a peer, advancing the head
// when the block extends it (optest_importBlock).
func (b *rethBackend) importBlock(ctx context.Context, payload json.RawMessage) (eth.PayloadStatusV1, error) {
	var status eth.PayloadStatusV1
	err := b.client.CallContext(ctx, &status, "optest_importBlock", payload)
	return status, err
}

// --- "Forkchoice requested sync to new head" log reproduction ---
//
// The in-process op-geth engine API logs "Forkchoice requested sync to new head" (with the head's
// block number) synchronously inside forkchoiceUpdated when the target head is one it only learned
// of via a prior newPayload it could not connect. The EL-sync tests assert on that exact line, and
// it must appear by the time the forkchoice call returns — so it is emitted here, in op-node's own
// call path via elSyncLogRPC, rather than from the asynchronous sync pump.

// elSyncLogRPC wraps the op-node -> reth engine RPC. It records the number of every payload op-node
// submits (engine_newPayload) and, when a forkchoice update (engine_forkchoiceUpdated) reports
// SYNCING for a head it has seen, emits the "Forkchoice requested sync to new head" log the geth
// engine API would.
type elSyncLogRPC struct {
	client.RPC
	b *rethBackend
}

func (r elSyncLogRPC) CallContext(ctx context.Context, result any, method string, args ...any) error {
	switch {
	case strings.HasPrefix(method, "engine_newPayload"):
		if len(args) > 0 {
			if payload, ok := args[0].(*eth.ExecutionPayload); ok {
				r.b.rememberPayload(payload.BlockHash, uint64(payload.BlockNumber))
			}
		}
		return r.RPC.CallContext(ctx, result, method, args...)
	case strings.HasPrefix(method, "engine_forkchoiceUpdated"):
		err := r.RPC.CallContext(ctx, result, method, args...)
		if err == nil {
			r.b.logSyncRequestIfPending(result, args)
		}
		return err
	default:
		return r.RPC.CallContext(ctx, result, method, args...)
	}
}

func (b *rethBackend) rememberPayload(hash common.Hash, number uint64) {
	b.mu.Lock()
	if b.syncSeen == nil {
		b.syncSeen = make(map[common.Hash]uint64)
	}
	b.syncSeen[hash] = number
	b.mu.Unlock()
}

// logSyncRequestIfPending emits the geth engine API's "Forkchoice requested sync to new head" line
// when a forkchoice update returned SYNCING for a head this engine only knows from a prior
// newPayload — the case where a real EL would begin snap-syncing towards it.
func (b *rethBackend) logSyncRequestIfPending(result any, args []any) {
	res, ok := result.(*eth.ForkchoiceUpdatedResult)
	if !ok || res.PayloadStatus.Status != eth.ExecutionSyncing || len(args) == 0 {
		return
	}
	state, ok := args[0].(*eth.ForkchoiceState)
	if !ok {
		return
	}
	b.mu.Lock()
	number, seen := b.syncSeen[state.HeadBlockHash]
	b.mu.Unlock()
	if !seen {
		return
	}
	b.log.Info("Forkchoice requested sync to new head", "number", number, "hash", state.HeadBlockHash)
}
