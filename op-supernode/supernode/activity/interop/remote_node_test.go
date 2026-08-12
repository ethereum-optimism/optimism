package interop

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	coreinterop "github.com/ethereum-optimism/optimism/op-core/interop"
	"github.com/ethereum-optimism/optimism/op-core/interop/messages"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/remote"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/remote/remotetest"
)

// newRemoteNode wires a deterministic test server to a real HTTPAdapter and the ingester,
// returning the ingester, the fake chain (for ExpectedChecksum), and a LogsDB to seal into.
func newRemoteNode(t *testing.T, cfg remotetest.Config) (*remoteNode, *remotetest.Chain, LogsDB) {
	node, chain, db, _ := newRemoteNodeInDir(t, cfg, t.TempDir())
	return node, chain, db
}

// newRemoteNodeInDir is newRemoteNode with an explicit LogsDB directory (so a test can
// reopen the same directory to simulate a restart) and a record of the `after` cursor
// every request carried, which is what the resume behaviour is actually about.
func newRemoteNodeInDir(t *testing.T, cfg remotetest.Config, dir string) (*remoteNode, *remotetest.Chain, LogsDB, *[]uint64) {
	chain := remotetest.New(cfg)
	var mu sync.Mutex
	asked := make([]uint64, 0, 8)
	inner := chain.Handler()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if after, err := strconv.ParseUint(r.URL.Query().Get("after"), 10, 64); err == nil {
			mu.Lock()
			asked = append(asked, after)
			mu.Unlock()
		}
		inner.ServeHTTP(w, r)
	}))
	t.Cleanup(srv.Close)

	db, err := openLogsDB(testLogger(), cfg.ChainID, dir)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	adapter := remote.NewHTTPAdapter(cfg.ChainID, srv.URL, srv.Client())
	node := &remoteNode{
		log: testLogger(), adapter: adapter, db: db,
		poll: defaultRemotePollInterval, maxPerCycle: defaultMaxBlocksPerCycle,
	}
	return node, chain, db, &asked
}

// scriptedAdapter is a remote.Adapter that returns exactly the blocks a test hands it,
// including responses no honest server would send, so the ingester's own guards can be
// exercised directly.
type scriptedAdapter struct {
	chainID eth.ChainID
	next    func(after uint64) (*remote.FinalizedBlock, bool, error)
}

func (a *scriptedAdapter) ChainID() eth.ChainID { return a.chainID }
func (a *scriptedAdapter) BlockTime() uint64    { return 2 }
func (a *scriptedAdapter) NextFinalized(_ context.Context, after uint64) (*remote.FinalizedBlock, bool, error) {
	return a.next(after)
}
func (a *scriptedAdapter) Close() error { return nil }

// TestRemoteNodeIngestAndContains exercises the full path against a real LogsDB: the
// ingester polls the remote node over HTTP, seals the fabricated initiating messages, and
// they become referenceable via the checksum the chain advertises (positive), while a
// wrong checksum is rejected (negative).
func TestRemoteNodeIngestAndContains(t *testing.T) {
	t.Parallel()
	const (
		activation   = uint64(1000)
		blockTime    = uint64(2)
		msgsPerBlock = 2
	)
	chainID := eth.ChainIDFromUInt64(8453)
	node, chain, db := newRemoteNode(t, remotetest.Config{
		ChainID: chainID, BlockTime: blockTime, MsgsPerBlock: msgsPerBlock, StartTimestamp: activation,
	})

	// Ingest two finalized blocks; the LogsDB resume cursor advances each time.
	for n := uint64(1); n <= 2; n++ {
		ingested, err := node.ingestOnce(context.Background())
		require.NoError(t, err)
		require.True(t, ingested)
		latest, ok := db.LatestSealedBlock()
		require.True(t, ok)
		require.Equal(t, n, latest.Number)
	}

	// Every fabricated message is referenceable via its advertised checksum.
	for n := uint64(1); n <= 2; n++ {
		ts := activation + n*blockTime
		for logIdx := uint32(0); logIdx < msgsPerBlock; logIdx++ {
			seal, err := db.Contains(messages.ContainsQuery{
				BlockNum:  n,
				LogIdx:    logIdx,
				Timestamp: ts,
				Checksum:  chain.ExpectedChecksum(n, logIdx),
			})
			require.NoError(t, err, "block %d log %d should be referenceable", n, logIdx)
			require.Equal(t, n, seal.Number)
			require.Equal(t, ts, seal.Timestamp)
		}
	}

	// A wrong checksum at a real (block, log) position is a conflict.
	_, err := db.Contains(messages.ContainsQuery{
		BlockNum:  1,
		LogIdx:    0,
		Timestamp: activation + blockTime,
		Checksum:  messages.MessageChecksum(common.HexToHash("0xbad")),
	})
	require.ErrorIs(t, err, coreinterop.ErrConflict)
}

// opSepoliaLikeHeight is a stand-in for a long-running chain's finalized head — the
// situation that makes replay-from-genesis impossible and anchoring mandatory.
const opSepoliaLikeHeight = uint64(47_300_000)

// TestRemoteNodeAnchorsAtHeight is the start-height anchoring case: a chain whose
// finalized head is tens of millions of blocks past genesis. The empty LogsDB accepts
// the first block served at whatever height it has, and the anchored (non-genesis-rooted)
// history is fully functional — Contains recomputes the checksum from the sealed record,
// and everything below the anchor reads as skipped rather than missing.
func TestRemoteNodeAnchorsAtHeight(t *testing.T) {
	t.Parallel()
	const (
		activation   = uint64(1000)
		blockTime    = uint64(2)
		msgsPerBlock = 2
	)
	chainID := eth.ChainIDFromUInt64(11155420)
	node, chain, db, asked := newRemoteNodeInDir(t, remotetest.Config{
		ChainID: chainID, BlockTime: blockTime, MsgsPerBlock: msgsPerBlock,
		StartTimestamp: activation, FirstBlock: opSepoliaLikeHeight,
	}, t.TempDir())

	_, hasBlocks := db.LatestSealedBlock()
	require.False(t, hasBlocks, "precondition: the LogsDB starts empty")

	ingested, err := node.ingestOnce(context.Background())
	require.NoError(t, err)
	require.True(t, ingested)
	require.Equal(t, []uint64{0}, *asked, "an empty LogsDB asks for the anchor with after=0")

	latest, ok := db.LatestSealedBlock()
	require.True(t, ok)
	require.Equal(t, opSepoliaLikeHeight, latest.Number, "the anchor is sealed at its real height")

	first, err := db.FirstSealedBlock()
	require.NoError(t, err)
	require.Equal(t, opSepoliaLikeHeight, first.Number, "anchored history starts at the anchor")

	// The anchor block's messages are referenceable: Contains recomputes the checksum
	// from (logHash, block number, log index, timestamp, chain ID), none of which
	// depend on the history below the anchor.
	anchorTS := activation + blockTime
	for logIdx := uint32(0); logIdx < msgsPerBlock; logIdx++ {
		seal, err := db.Contains(messages.ContainsQuery{
			BlockNum:  opSepoliaLikeHeight,
			LogIdx:    logIdx,
			Timestamp: anchorTS,
			Checksum:  chain.ExpectedChecksum(opSepoliaLikeHeight, logIdx),
		})
		require.NoError(t, err, "anchor block log %d must be referenceable", logIdx)
		require.Equal(t, opSepoliaLikeHeight, seal.Number)
		require.Equal(t, anchorTS, seal.Timestamp)
	}

	// A wrong checksum at a real position is still a conflict, so anchoring did not
	// weaken verification.
	_, err = db.Contains(messages.ContainsQuery{
		BlockNum: opSepoliaLikeHeight, LogIdx: 0, Timestamp: anchorTS,
		Checksum: messages.MessageChecksum(common.HexToHash("0xbad")),
	})
	require.ErrorIs(t, err, coreinterop.ErrConflict)

	// History below the anchor was never ingested and is reported as such.
	_, err = db.Contains(messages.ContainsQuery{
		BlockNum: opSepoliaLikeHeight - 1, LogIdx: 0, Timestamp: anchorTS - blockTime,
		Checksum: chain.ExpectedChecksum(opSepoliaLikeHeight-1, 0),
	})
	require.ErrorIs(t, err, coreinterop.ErrSkipped)

	// Ingestion continues contiguously from the anchor.
	ingested, err = node.ingestOnce(context.Background())
	require.NoError(t, err)
	require.True(t, ingested)
	latest, ok = db.LatestSealedBlock()
	require.True(t, ok)
	require.Equal(t, opSepoliaLikeHeight+1, latest.Number)
	require.Equal(t, []uint64{0, opSepoliaLikeHeight}, *asked,
		"once anchored, the cursor is the latest sealed block")
}

// TestRemoteNodeContiguityAfterAnchor pins the other half of the rule: the height
// freedom is exhausted by the anchor. Once history exists, a block at any height other
// than latest+1 is rejected and nothing is sealed.
func TestRemoteNodeContiguityAfterAnchor(t *testing.T) {
	t.Parallel()
	const blockTime = uint64(2)
	chainID := eth.ChainIDFromUInt64(11155420)
	chain := remotetest.New(remotetest.Config{
		ChainID: chainID, BlockTime: blockTime, StartTimestamp: 1000,
		FirstBlock: opSepoliaLikeHeight,
	})

	db, err := openLogsDB(testLogger(), chainID, t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	// The scripted adapter answers the anchor request honestly, then jumps a block.
	served := chain.Block(opSepoliaLikeHeight)
	adapter := &scriptedAdapter{chainID: chainID, next: func(after uint64) (*remote.FinalizedBlock, bool, error) {
		if after == 0 {
			return served, true, nil
		}
		// A gap: skipping straight from the anchor to anchor+2.
		return chain.Block(after + 2), true, nil
	}}
	node := &remoteNode{log: testLogger(), adapter: adapter, db: db, poll: defaultRemotePollInterval}

	ingested, err := node.ingestOnce(context.Background())
	require.NoError(t, err)
	require.True(t, ingested)

	_, err = node.ingestOnce(context.Background())
	require.ErrorContains(t, err, "non-contiguous")
	latest, ok := db.LatestSealedBlock()
	require.True(t, ok)
	require.Equal(t, opSepoliaLikeHeight, latest.Number, "a rejected block must not be sealed")

	// The same guard applies to a block below the head, not just above it.
	adapter.next = func(after uint64) (*remote.FinalizedBlock, bool, error) {
		return chain.Block(after), true, nil
	}
	_, err = node.ingestOnce(context.Background())
	require.ErrorContains(t, err, "non-contiguous")
}

// TestRemoteNodeAnchorSurvivesRestart reopens the same LogsDB directory with a fresh
// ingester: the anchor is durable, so the restarted node resumes from LatestSealedBlock
// instead of re-anchoring somewhere else.
func TestRemoteNodeAnchorSurvivesRestart(t *testing.T) {
	t.Parallel()
	const (
		activation = uint64(1000)
		blockTime  = uint64(2)
	)
	chainID := eth.ChainIDFromUInt64(11155420)
	dir := filepath.Join(t.TempDir(), "logsdb")
	cfg := remotetest.Config{
		ChainID: chainID, BlockTime: blockTime, StartTimestamp: activation,
		FirstBlock: opSepoliaLikeHeight,
	}

	node, _, db, asked := newRemoteNodeInDir(t, cfg, dir)
	for i := 0; i < 2; i++ {
		ingested, err := node.ingestOnce(context.Background())
		require.NoError(t, err)
		require.True(t, ingested)
	}
	require.Equal(t, []uint64{0, opSepoliaLikeHeight}, *asked)
	require.NoError(t, db.Close())

	// Restart: same directory, new DB handle, new ingester.
	node2, chain2, db2, asked2 := newRemoteNodeInDir(t, cfg, dir)
	latest, ok := db2.LatestSealedBlock()
	require.True(t, ok, "sealed history must survive the restart")
	require.Equal(t, opSepoliaLikeHeight+1, latest.Number)
	first, err := db2.FirstSealedBlock()
	require.NoError(t, err)
	require.Equal(t, opSepoliaLikeHeight, first.Number, "the anchor is still the start of history")

	ingested, err := node2.ingestOnce(context.Background())
	require.NoError(t, err)
	require.True(t, ingested)
	require.Equal(t, []uint64{opSepoliaLikeHeight + 1}, *asked2,
		"a restarted node resumes from its latest sealed block, never re-anchoring")

	latest, ok = db2.LatestSealedBlock()
	require.True(t, ok)
	require.Equal(t, opSepoliaLikeHeight+2, latest.Number)

	// Messages sealed before the restart are still referenceable afterwards.
	seal, err := db2.Contains(messages.ContainsQuery{
		BlockNum:  opSepoliaLikeHeight,
		LogIdx:    0,
		Timestamp: activation + blockTime,
		Checksum:  chain2.ExpectedChecksum(opSepoliaLikeHeight, 0),
	})
	require.NoError(t, err)
	require.Equal(t, opSepoliaLikeHeight, seal.Number)
}

// TestRemoteNodeIngestCycleDrains covers catch-up: one cycle must drain every block the
// adapter is willing to serve, not one per tick. At one block per poll interval an
// ingester could only match the remote chain's production rate and would never close a
// gap opened by downtime.
func TestRemoteNodeIngestCycleDrains(t *testing.T) {
	t.Parallel()
	const (
		blockTime = uint64(2)
		backlog   = uint64(10)
	)
	chainID := eth.ChainIDFromUInt64(11155420)
	head := opSepoliaLikeHeight + backlog - 1
	node, _, db, asked := newRemoteNodeInDir(t, remotetest.Config{
		ChainID: chainID, BlockTime: blockTime, StartTimestamp: 1000,
		FirstBlock: opSepoliaLikeHeight, Head: head,
	}, t.TempDir())

	require.NoError(t, node.ingestCycle(context.Background()))

	latest, ok := db.LatestSealedBlock()
	require.True(t, ok)
	require.Equal(t, head, latest.Number, "one cycle must drain the whole backlog")
	// backlog ingests plus the final request that reported nothing new.
	require.Len(t, *asked, int(backlog)+1)

	// A cycle with nothing to do is a single request and no error.
	before := len(*asked)
	require.NoError(t, node.ingestCycle(context.Background()))
	require.Equal(t, head, mustLatest(t, db).Number)
	require.Len(t, *asked, before+1)
}

// TestRemoteNodeIngestCycleRespectsCap keeps a cycle bounded: a chain that is arbitrarily
// far ahead must not be drained in one unbounded burst, or shutdown latency and adapter
// load become unbounded with it. Progress simply continues on the next tick.
func TestRemoteNodeIngestCycleRespectsCap(t *testing.T) {
	t.Parallel()
	chainID := eth.ChainIDFromUInt64(11155420)
	node, _, db, _ := newRemoteNodeInDir(t, remotetest.Config{
		ChainID: chainID, BlockTime: 2, StartTimestamp: 1000,
		FirstBlock: opSepoliaLikeHeight, // unbounded: always another block
	}, t.TempDir())
	node.maxPerCycle = 4

	require.NoError(t, node.ingestCycle(context.Background()))
	require.Equal(t, opSepoliaLikeHeight+3, mustLatest(t, db).Number, "capped at 4 blocks")

	require.NoError(t, node.ingestCycle(context.Background()))
	require.Equal(t, opSepoliaLikeHeight+7, mustLatest(t, db).Number, "the next cycle resumes")
}

// TestRemoteNodeIngestCycleStopsOnCancel makes sure a cancelled context ends the cycle
// promptly rather than running to the cap.
func TestRemoteNodeIngestCycleStopsOnCancel(t *testing.T) {
	t.Parallel()
	chainID := eth.ChainIDFromUInt64(11155420)
	node, _, db, _ := newRemoteNodeInDir(t, remotetest.Config{
		ChainID: chainID, BlockTime: 2, StartTimestamp: 1000,
		FirstBlock: opSepoliaLikeHeight,
	}, t.TempDir())

	ctx, cancel := context.WithCancel(context.Background())
	ingested, err := node.ingestOnce(ctx)
	require.NoError(t, err)
	require.True(t, ingested)
	cancel()

	require.ErrorIs(t, node.ingestCycle(ctx), context.Canceled)
	require.Equal(t, opSepoliaLikeHeight, mustLatest(t, db).Number)
}

func mustLatest(t *testing.T, db LogsDB) eth.BlockID {
	t.Helper()
	latest, ok := db.LatestSealedBlock()
	require.True(t, ok)
	return latest
}

func TestAddRemoteNode(t *testing.T) {
	// newInteropTestHarness calls t.Parallel() internally.
	h := newInteropTestHarness(t).WithActivation(1000).WithChain(10, nil).Build()
	require.NotNil(t, h.interop)

	remoteID := eth.ChainIDFromUInt64(8453)
	adapter := remote.NewHTTPAdapter(remoteID, "http://example.invalid", nil)

	require.NoError(t, h.interop.AddRemoteNode(adapter))
	require.Contains(t, h.interop.remoteNodes, remoteID)
	require.Contains(t, h.interop.logsDBs, remoteID, "remote logsDB must be registered for executing-message validation to read it")

	// Duplicate remote registration fails.
	require.Error(t, h.interop.AddRemoteNode(adapter))

	// A driven chain cannot also be a remote node.
	require.Error(t, h.interop.AddRemoteNode(remote.NewHTTPAdapter(eth.ChainIDFromUInt64(10), "http://example.invalid", nil)))
}

// TestVerifyExecutingMessageReferencesRemoteNode is the end-to-end check: a driven chain
// references a remote node's initiating message as an executing message, through the real
// verifyExecutingMessage path (including the remote block-time fallback). The remote
// node's messages arrive over HTTP from the test server.
func TestVerifyExecutingMessageReferencesRemoteNode(t *testing.T) {
	// newInteropTestHarness calls t.Parallel() internally.
	const (
		drivenChain = 10
		remoteChain = 8453
		activation  = uint64(1000)
		blockTime   = uint64(2)
	)
	h := newInteropTestHarness(t).WithActivation(activation).WithChain(drivenChain, nil).Build()
	require.NotNil(t, h.interop)

	remoteID := eth.ChainIDFromUInt64(remoteChain)
	chain := remotetest.New(remotetest.Config{
		ChainID: remoteID, BlockTime: blockTime, MsgsPerBlock: 1, StartTimestamp: activation,
	})
	srv := httptest.NewServer(chain.Handler())
	defer srv.Close()
	require.NoError(t, h.interop.AddRemoteNode(remote.NewHTTPAdapter(remoteID, srv.URL, srv.Client())))

	// Ingest one finalized block from the remote node (also populates the adapter's
	// cached block time, used by the activation invariant below).
	node := h.interop.remoteNodes[remoteID]
	require.NotNil(t, node)
	ingested, err := node.ingestOnce(context.Background())
	require.NoError(t, err)
	require.True(t, ingested)

	const initBlock, initLogIdx = uint64(1), uint32(0)
	initTimestamp := activation + blockTime // block 1 timestamp = 1002

	execMsg := &messages.ExecutingMessage{
		ChainID:   remoteID,
		BlockNum:  initBlock,
		LogIdx:    initLogIdx,
		Timestamp: initTimestamp,
		Checksum:  chain.ExpectedChecksum(initBlock, initLogIdx),
	}
	drivenID := eth.ChainIDFromUInt64(drivenChain)

	// Positive: valid executing message referencing the remote node passes.
	require.NoError(t, h.interop.verifyExecutingMessage(drivenID, initTimestamp+50, 0, execMsg, nil),
		"valid executing message referencing a remote node must pass")

	// Negative: a wrong checksum is a conflict.
	bad := *execMsg
	bad.Checksum = messages.MessageChecksum(common.HexToHash("0xdeadbeef"))
	err = h.interop.verifyExecutingMessage(drivenID, initTimestamp+50, 0, &bad, nil)
	require.ErrorIs(t, err, coreinterop.ErrConflict)

	// Negative: referencing a block the remote node has not ingested yet fails.
	future := *execMsg
	future.BlockNum = 99
	future.Timestamp = activation + 99*blockTime
	future.Checksum = chain.ExpectedChecksum(99, 0)
	require.Error(t, h.interop.verifyExecutingMessage(drivenID, future.Timestamp+50, 0, &future, nil),
		"referencing a not-yet-ingested remote block must fail")
}

// TestVerifyExecutingMessageReferencesAnchoredRemoteNode is the previous test's case for
// an anchored remote chain: a driven chain references an initiating message in a block
// tens of millions of blocks past the remote chain's genesis, which is the only shape
// available when the remote chain is a live public network. Nothing in the verification
// path depends on the remote history being genesis-rooted.
func TestVerifyExecutingMessageReferencesAnchoredRemoteNode(t *testing.T) {
	// newInteropTestHarness calls t.Parallel() internally.
	const (
		drivenChain = 10
		remoteChain = 11155420
		activation  = uint64(1000)
		blockTime   = uint64(2)
	)
	h := newInteropTestHarness(t).WithActivation(activation).WithChain(drivenChain, nil).Build()
	require.NotNil(t, h.interop)

	remoteID := eth.ChainIDFromUInt64(remoteChain)
	chain := remotetest.New(remotetest.Config{
		ChainID: remoteID, BlockTime: blockTime, MsgsPerBlock: 1,
		StartTimestamp: activation, FirstBlock: opSepoliaLikeHeight,
	})
	srv := httptest.NewServer(chain.Handler())
	defer srv.Close()
	require.NoError(t, h.interop.AddRemoteNode(remote.NewHTTPAdapter(remoteID, srv.URL, srv.Client())))

	node := h.interop.remoteNodes[remoteID]
	require.NotNil(t, node)
	require.Equal(t, defaultMaxBlocksPerCycle, node.maxPerCycle,
		"AddRemoteNode must configure the per-cycle cap")
	// Drain a small backlog in one cycle, the way a catching-up node does.
	require.NoError(t, node.ingestCycle(context.Background()))
	require.Equal(t, opSepoliaLikeHeight+uint64(defaultMaxBlocksPerCycle)-1,
		mustLatest(t, h.interop.logsDBs[remoteID]).Number)

	const initLogIdx = uint32(0)
	initBlock := opSepoliaLikeHeight + 3
	initTimestamp := activation + 4*blockTime // 4th block served

	execMsg := &messages.ExecutingMessage{
		ChainID:   remoteID,
		BlockNum:  initBlock,
		LogIdx:    initLogIdx,
		Timestamp: initTimestamp,
		Checksum:  chain.ExpectedChecksum(initBlock, initLogIdx),
	}
	drivenID := eth.ChainIDFromUInt64(drivenChain)

	require.NoError(t, h.interop.verifyExecutingMessage(drivenID, initTimestamp+50, 0, execMsg, nil),
		"an executing message referencing anchored remote history must verify")

	bad := *execMsg
	bad.Checksum = messages.MessageChecksum(common.HexToHash("0xdeadbeef"))
	require.ErrorIs(t, h.interop.verifyExecutingMessage(drivenID, initTimestamp+50, 0, &bad, nil),
		coreinterop.ErrConflict)

	// A message from below the anchor cannot be verified: that history was never
	// ingested, and the verifier must not accept it on faith.
	belowAnchor := *execMsg
	belowAnchor.BlockNum = opSepoliaLikeHeight - 1
	belowAnchor.Checksum = chain.ExpectedChecksum(belowAnchor.BlockNum, 0)
	require.Error(t, h.interop.verifyExecutingMessage(drivenID, initTimestamp+50, 0, &belowAnchor, nil),
		"referencing pre-anchor history must fail")
}
