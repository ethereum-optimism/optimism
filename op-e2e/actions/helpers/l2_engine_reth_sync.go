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
// sync pump. The pump watches the engine's sync target (`optest_syncTarget`, the head a forkchoice
// update reported SYNCING for) and, while one is set, copies the missing blocks from the peer engine
// one validated `optest_importBlock` at a time — reproducing what op-geth's snap sync does, without
// the p2p. Because the pump only runs once the engine has actually reported SYNCING (which op-node
// triggers only during genuine EL sync, never for a CL-mode gapped payload), it activates exactly
// when a real EL would snap-sync and stays idle otherwise.

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

// addPeer records a symmetric peering with `peer` and starts a sync pump that backfills this engine
// from it. Peering is bidirectional (as devp2p peering is) so both engines report a non-zero
// PeerCount, which the EL-sync tests wait on before inserting the unsafe head.
func (b *rethBackend) addPeer(peer *rethBackend) {
	b.mu.Lock()
	_, known := b.peers[peer.id]
	b.peers[peer.id] = peer
	b.mu.Unlock()

	peer.mu.Lock()
	peer.peers[b.id] = b
	peer.mu.Unlock()

	if !known {
		b.startPump(peer)
	}
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

func (b *rethBackend) startPump(peer *rethBackend) {
	ctx, cancel := context.WithCancel(context.Background())
	b.mu.Lock()
	b.pumpCancels = append(b.pumpCancels, cancel)
	b.mu.Unlock()

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

// pumpOnce performs one backfill pass. If the engine has reported a sync target it cannot reach on
// its own — the head a real EL would snap-sync towards — it copies the still-missing blocks in from
// the peer, one validated newPayload at a time, in ascending order so each block's parent is
// already present. It deliberately never moves the safe/finalized pointers (those wait for op-node's
// forkchoice updates), reproducing exactly the state op-geth reaches after backfilling under a
// SYNCING forkchoice: the chain is present, but nothing is yet marked safe.
func (b *rethBackend) pumpOnce(ctx context.Context, peer *rethBackend) {
	target, err := b.syncTarget(ctx)
	if err != nil || target == (common.Hash{}) {
		return
	}
	localHead, err := b.blockNumber(ctx)
	if err != nil {
		return
	}
	peerHead, err := peer.blockNumber(ctx)
	if err != nil {
		return
	}
	for n := localHead + 1; n <= peerHead; n++ {
		if ctx.Err() != nil {
			return
		}
		payload, err := peer.blockPayload(ctx, n)
		if err != nil || payload == nil {
			return
		}
		if err := b.importBlock(ctx, payload); err != nil {
			return
		}
	}
}

// shutdown stops every sync pump, removes the engine from the peer registry, and closes the
// subprocess. It is safe to call more than once (t.Cleanup plus an explicit L2Engine.Close).
func (b *rethBackend) shutdown() {
	b.shutdownOnce.Do(func() {
		b.mu.Lock()
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

// importBlock validates, executes, and commits a block obtained from a peer (optest_importBlock).
func (b *rethBackend) importBlock(ctx context.Context, payload json.RawMessage) error {
	var status json.RawMessage
	return b.client.CallContext(ctx, &status, "optest_importBlock", payload)
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
