package silhouette

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"

	altda "github.com/ethereum-optimism/optimism/op-alt-da"
	"github.com/ethereum-optimism/optimism/op-node/metrics"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/attributes"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-node/rollup/driver"
	"github.com/ethereum-optimism/optimism/op-node/rollup/engine"
	"github.com/ethereum-optimism/optimism/op-node/rollup/finality"
	"github.com/ethereum-optimism/optimism/op-node/rollup/status"
	"github.com/ethereum-optimism/optimism/op-node/rollup/sync"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/event"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
)

// THE HARNESS SWAP (G3 gate 1).
//
// G2's pipeline test stopped at the engine: the shim did not exist, so a `factsL2` stub played its
// part. This is the same test with the stub gone and a REAL rollup node standing in its place —
// event system, engine controller, attributes handler, pipeline deriver, sync deriver, finalizer,
// status tracker — driving the shim over JSON-RPC.
//
// Every component below is stock op-node. The ASSEMBLY is local, and it is a deliberate copy of
// op-e2e/actions/helpers/l2_verifier.go's assembly (which is itself test glue for exactly this
// purpose): that file cannot be reused directly because it does not pass a derive.PipelineOption
// through, and the whole point here is the injected data source. What is emphatically not local is
// anything that talks to the shim: the engine controller decides when to forkchoice-update, the
// attributes handler decides what to build, and sources.EngineClient makes every call.
//
// The synchronous event executor (event.NewGlobalSynchronous) is what makes this a test rather than
// a race: `Drain()` runs the node's event graph to quiescence, so an assertion after it sees a
// settled node.

// verifierEnv is a stock rollup node whose execution client is the shim.
type verifierEnv struct {
	*shimEnv
	sys      event.System
	drainer  event.Drainer
	ec       *engine.EngineController
	pipeline *derive.DerivationPipeline
	trigger  event.Emitter
	idle     bool
}

func (se *shimEnv) newVerifier(t *testing.T) *verifierEnv {
	t.Helper()
	logger := testlog.Logger(t, 3)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	executor := event.NewGlobalSynchronous(ctx)
	sys := event.NewSystem(logger, executor)
	t.Cleanup(sys.Stop)
	opts := event.WithEmitLimiter(rate.Limit(100_000), 100_000, func() {
		t.Fatal("events must not hot-loop")
	})

	syncCfg := &sync.Config{SyncMode: sync.CLSync}
	ec := engine.NewEngineController(ctx, se.eng, logger, metrics.NoopMetrics, se.rollup, syncCfg,
		se.l1, sys.Register("engine-controller", nil, opts), nil)

	fin := finality.NewFinalizer(ctx, logger, se.rollup, nil, se.l1, ec)
	sys.Register("finalizer", fin, opts)

	attrHandler := attributes.NewAttributesHandler(logger, se.rollup, ctx, se.eng, ec)
	sys.Register("attributes-handler", attrHandler, opts)
	ec.SetAttributesResetter(attrHandler)

	pipeline := derive.NewDerivationPipeline(
		logger, se.rollup,
		staticDepSet{chains: []eth.ChainID{eth.ChainIDFromBig(se.rollup.L2ChainID)}},
		se.l1, se.blobs,
		altda.NewAltDA(logger, altda.CLIConfig{}, altda.Config{}, &altda.NoopMetrics{}),
		se.eng, &testutils.TestDerivationMetrics{}, sepoliaChainConfig(),
		derive.WithDataSource(se.src),
	)
	pipelineDeriver := derive.NewPipelineDeriver(ctx, pipeline)
	sys.Register("pipeline", pipelineDeriver, opts)
	ec.SetPipelineResetter(pipelineDeriver)

	trigger := sys.Register("test-trigger", nil, opts)
	statusTracker := status.NewStatusTracker(logger, &testutils.TestDerivationMetrics{})
	sys.Register("status", statusTracker, opts)
	ec.SetCrossUpdateHandler(statusTracker)

	steps := &triggerStepDeriver{emitter: trigger}
	syncDeriver := &driver.SyncDeriver{
		Derivation:     pipeline,
		SafeHeadNotifs: noopSafeHeadListener{},
		Engine:         ec,
		SyncCfg:        syncCfg,
		Config:         se.rollup,
		L1:             se.l1,
		L2:             se.eng,
		Log:            logger,
		Ctx:            ctx,
		StepDeriver:    steps,
	}
	ec.SyncDeriver = syncDeriver
	sys.Register("sync", syncDeriver, opts)
	sys.Register("engine", ec, opts)

	ve := &verifierEnv{shimEnv: se, sys: sys, drainer: executor, ec: ec, pipeline: pipeline, trigger: trigger, idle: true}
	sys.Register("verifier", ve, opts)
	return ve
}

// OnEvent makes the harness a deriver, so that a critical error is a test failure rather than a log
// line, and so idleness is observable.
func (v *verifierEnv) OnEvent(ctx context.Context, ev event.Event) bool {
	switch x := ev.(type) {
	case rollup.CriticalErrorEvent:
		panic(fmt.Errorf("derivation failed critically: %w", x.Err))
	case rollup.EngineTemporaryErrorEvent:
		v.t.Logf("engine temporary error: %v", x.Err)
	case rollup.ResetEvent:
		v.t.Logf("pipeline reset: %v", x.Err)
	case derive.DeriverIdleEvent:
		v.idle = true
	case derive.PipelineStepEvent:
		v.idle = false
	default:
		return false
	}
	return true
}

// runToQuiescence drives the node the way the real driver does — a step signal, then the whole event
// graph run to completion — until the safe head stops moving.
func (v *verifierEnv) runToQuiescence(t *testing.T, rounds int) {
	t.Helper()
	ctx := context.Background()
	safeHead := func() uint64 { return v.ec.SafeL2Head().Number }
	for i := 0; i < rounds; i++ {
		before := safeHead()
		v.trigger.Emit(ctx, driver.StepEvent{})
		require.NoError(t, v.drainer.Drain(), "the node's event graph must run to quiescence")
		if safeHead() == before && v.idle && i > 2 {
			return
		}
	}
}

// triggerStepDeriver is the step scheduler as a test drives it: a requested step is emitted
// immediately rather than queued behind a timer.
type triggerStepDeriver struct{ emitter event.Emitter }

func (s *triggerStepDeriver) NextStep() <-chan struct{}         { return nil }
func (s *triggerStepDeriver) NextDelayedStep() <-chan time.Time { return nil }
func (s *triggerStepDeriver) RequestStep(ctx context.Context, resetBackoff bool) {
	s.emitter.Emit(ctx, driver.StepEvent{})
}
func (s *triggerStepDeriver) AttemptStep(ctx context.Context)      {}
func (s *triggerStepDeriver) ResetStepBackoff(ctx context.Context) {}
func (s *triggerStepDeriver) AttachEmitter(em event.Emitter)       { s.emitter = em }

type noopSafeHeadListener struct{}

func (noopSafeHeadListener) Enabled() bool { return false }
func (noopSafeHeadListener) SafeHeadUpdated(eth.L2BlockRef, eth.BlockID) error {
	return nil
}
func (noopSafeHeadListener) SafeHeadReset(eth.L2BlockRef) error { return nil }
func (noopSafeHeadListener) Close() error                       { return nil }

// TestStockNodeDrivesTheShim is the harness swap: G2's gate-2 derivation, with a real rollup node
// and the shim as its execution client.
//
// The assertions are the ones that would be impossible if any part of the claim were false: the
// node's own safe head is the proven chain's REAL hash, the labels move down the ladder through
// ordinary forkchoice calls, and the settlement-facing output roots come back out of the shim.
func TestStockNodeDrivesTheShim(t *testing.T) {
	e := newTestEnv(t, l1GenesisNum+4).withRealFeeScalars(t)
	spec := e.goodBatch()
	batch := e.buildBatch(spec)
	e.plant(batch, spec)

	se := e.newShim(t)
	v := se.newVerifier(t)
	v.runToQuiescence(t, 400)

	ctx := context.Background()
	last := batch.Blocks[len(batch.Blocks)-1]

	// THE LADDER MOVED, and it moved to the real hashes. Nothing here computed a label: the engine
	// controller did, out of forkchoice calls the shim answered.
	safe := v.ec.SafeL2Head()
	require.Equal(t, last.Number, safe.Number, "the node's safe head must reach the proven head")
	require.Equal(t, last.Hash, safe.Hash, "and it must BE the hash the proof committed to")
	require.Equal(t, last.Hash, v.ec.UnsafeL2Head().Hash)
	cursors := se.shim.facts.Cursors()
	require.Equal(t, last.Hash, cursors.Unsafe.Hash, "the shim's own cursors must agree with the node's")
	require.Equal(t, last.Hash, cursors.Safe.Hash)

	// Every proven block is on the node's chain, by its real hash, with its real parent.
	for i, blk := range batch.Blocks {
		ref, err := se.eng.L2BlockRefByNumber(ctx, blk.Number)
		require.NoError(t, err)
		require.Equal(t, blk.Hash, ref.Hash, "block %d", i)
		wantParent := e.rollup.Genesis.L2.Hash
		if i > 0 {
			wantParent = batch.Blocks[i-1].Hash
		}
		require.Equal(t, wantParent, ref.ParentHash, "block %d parent", i)

		out, err := se.eng.OutputV0AtBlockNumber(ctx, blk.Number)
		require.NoError(t, err)
		require.Equal(t, blk.OutputRoot(), common.Hash(eth.OutputRoot(out)),
			"block %d: a stock node reading a stock client must see the settlement value", i)
	}

	// And the batch's claimed newOutputRoot is what the node's safe head yields.
	out, err := se.eng.OutputV0AtBlock(ctx, safe.Hash)
	require.NoError(t, err)
	require.Equal(t, batch.NewOutputRoot, common.Hash(eth.OutputRoot(out)))

	_, halted := se.shim.Halted()
	require.False(t, halted, "a clean run must not trip the honesty assertion")
}

// TestStockNodeReDerivesIdenticallyAfterAReset is the reorg half of gate 1 and gate 4's re-derivation
// requirement, driven by the node rather than by the test.
//
// A pipeline reset makes the node walk back through the shim's served headers (FindL2Heads), rewinds
// the transcoder's chaining state with it (G2 D5), and re-derives. The chain that comes back must be
// the same chain — same hashes at the same heights — because both the source and the shim are pure
// functions of L1 and the facts.
func TestStockNodeReDerivesIdenticallyAfterAReset(t *testing.T) {
	e := newTestEnv(t, l1GenesisNum+4).withRealFeeScalars(t)
	spec := e.goodBatch()
	batch := e.buildBatch(spec)
	e.plant(batch, spec)

	se := e.newShim(t)
	v := se.newVerifier(t)
	v.runToQuiescence(t, 400)

	last := batch.Blocks[len(batch.Blocks)-1]
	before := v.ec.SafeL2Head()
	require.Equal(t, last.Hash, before.Hash)

	// Forget every rendering first. That makes the re-derivation take the DETERMINISTIC
	// RECONSTRUCTION path — the shim rebuilds each block's body from the frozen config and the
	// rendered origin rather than replaying what it stored — and consolidation then compares that
	// reconstruction against the attributes the pipeline re-derives. A single wrong byte in the
	// rebuilt L1-info transaction would show up here as the node reorging out its own chain, which is
	// exactly what a restarted verifier must not do.
	e.facts.DropRenderingsAbove(0)
	_, held := e.facts.Rendering(last.Hash)
	require.False(t, held)

	// Reset the pipeline through the stock path — the one admin_resetDerivationPipeline uses, and the
	// one an L1 reorg ends up on. The reset walk that follows is FindL2Heads over the shim's served
	// headers, and the transcoder's chaining state rewinds with it (G2 D5).
	v.pipeline.Reset()
	v.runToQuiescence(t, 400)

	after := v.ec.SafeL2Head()
	require.Equal(t, before, after, "re-derivation after a reset must land on the same safe head")
	for _, blk := range batch.Blocks {
		fact, ok := e.facts.ByNumber(blk.Number)
		require.True(t, ok, "block %d must still be a fact after a reset", blk.Number)
		require.Equal(t, blk.Hash, fact.Hash)
	}
	_, halted := se.shim.Halted()
	require.False(t, halted, "a reset is not a fabrication")
}

// TestStockNodeStopsAtTheProvenFrontier: with no further proof batch on L1, the node derives to the
// proven head and STOPS there. It does not force-generate (the window has not expired), it does not
// halt, and it never asks the shim for a block the facts do not cover.
func TestStockNodeStopsAtTheProvenFrontier(t *testing.T) {
	e := newTestEnv(t, l1GenesisNum+4).withRealFeeScalars(t)
	spec := e.goodBatch()
	batch := e.buildBatch(spec)
	e.plant(batch, spec)

	se := e.newShim(t)
	v := se.newVerifier(t)
	v.runToQuiescence(t, 400)

	last := batch.Blocks[len(batch.Blocks)-1]
	require.Equal(t, last.Number, v.ec.SafeL2Head().Number)

	// Advance L1 well past the batch — but not a full sequencing window — and drive again.
	e.l1.head += 20
	v.runToQuiescence(t, 400)
	require.Equal(t, last.Number, v.ec.SafeL2Head().Number,
		"derivation must stop at the proven frontier: there is no more proof and no forced block due")
	head, ok := e.facts.Head()
	require.True(t, ok)
	require.Equal(t, last.Number, head.Number, "and no fact may appear for a block nothing proved")
	_, halted := se.shim.Halted()
	require.False(t, halted)
}
