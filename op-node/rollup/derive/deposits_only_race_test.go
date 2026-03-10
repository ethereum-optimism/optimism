package derive

import (
	"context"
	"io"
	"math/big"
	"math/rand"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"

	altda "github.com/ethereum-optimism/optimism/op-alt-da"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/event"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
)

// --- Stubs for NewDerivationPipeline dependencies ---

// stubBlobsFetcher satisfies L1BlobsFetcher for testing.
type stubBlobsFetcher struct{}

func (s *stubBlobsFetcher) GetBlobsByHash(_ context.Context, _ uint64, _ []common.Hash) ([]*eth.Blob, error) {
	return nil, nil
}

// stubAltDAFetcher satisfies AltDAInputFetcher for testing.
type stubAltDAFetcher struct{}

func (s *stubAltDAFetcher) GetInput(_ context.Context, _ altda.L1Fetcher, _ altda.CommitmentData, _ eth.L1BlockRef) (eth.Data, error) {
	return nil, nil
}
func (s *stubAltDAFetcher) AdvanceL1Origin(_ context.Context, _ altda.L1Fetcher, _ eth.BlockID) error {
	return nil
}
func (s *stubAltDAFetcher) Reset(_ context.Context, _ eth.L1BlockRef, _ eth.SystemConfig) error {
	return io.EOF
}

// stubL2Source satisfies L2Source for testing.
type stubL2Source struct {
	testutils.MockL2Client
}

func (s *stubL2Source) PayloadByHash(_ context.Context, _ common.Hash) (*eth.ExecutionPayloadEnvelope, error) {
	return nil, nil
}
func (s *stubL2Source) PayloadByNumber(_ context.Context, _ uint64) (*eth.ExecutionPayloadEnvelope, error) {
	return nil, nil
}

// makeTestRollupConfig creates a rollup config with Holocene active at genesis.
// Holocene must be active because DepositsOnlyPayloadAttributesRequestEvent
// is a Holocene-era feature, and FlushChannel (called by DepositsOnlyAttributes)
// requires the Holocene BatchStage.
func makeTestRollupConfig() *rollup.Config {
	zero := uint64(0)
	return &rollup.Config{
		BlockTime:              2,
		SeqWindowSize:          120,
		L1ChainID:              big.NewInt(101),
		L2ChainID:              big.NewInt(102),
		DepositContractAddress: common.Address{0xbb},
		L1SystemConfigAddress:  common.Address{0xcc},
		Genesis: rollup.Genesis{
			L1:     eth.BlockID{Hash: common.Hash{0x01}, Number: 0},
			L2:     eth.BlockID{Hash: common.Hash{0x02}, Number: 0},
			L2Time: 1000,
			SystemConfig: eth.SystemConfig{
				BatcherAddr: common.Address{42},
				GasLimit:    20_000_000,
			},
		},
		// Activate all forks at genesis so Holocene stages are used.
		RegolithTime:  &zero,
		CanyonTime:    &zero,
		DeltaTime:     &zero,
		EcotoneTime:   &zero,
		FjordTime:     &zero,
		GraniteTime:   &zero,
		HoloceneTime:  &zero,
	}
}

// makeTestAttribs creates test AttributesWithParent for the race condition tests.
func makeTestAttribs(rng *rand.Rand) *AttributesWithParent {
	parent := testutils.RandomL2BlockRef(rng)
	derivedFrom := testutils.RandomBlockRef(rng)

	return &AttributesWithParent{
		Attributes: &eth.PayloadAttributes{
			Timestamp:    eth.Uint64Quantity(parent.Time + 2),
			Transactions: []eth.Data{eth.Data("deposit-tx")},
			NoTxPool:     true,
		},
		Parent:      parent,
		Concluding:  false,
		DerivedFrom: derivedFrom,
	}
}

// newFullPipeline creates a DerivationPipeline via NewDerivationPipeline with
// all stages wired up. The pipeline is in its initial state (resetting=0).
func newFullPipeline(t *testing.T, logger log.Logger, cfg *rollup.Config) (*DerivationPipeline, *testutils.MockL1Source) {
	t.Helper()

	l1Fetcher := &testutils.MockL1Source{}
	pipeline := NewDerivationPipeline(
		logger, cfg, nil, // depSet
		l1Fetcher, &stubBlobsFetcher{}, &stubAltDAFetcher{}, &stubL2Source{},
		&testutils.TestDerivationMetrics{}, false, // managedBySupervisor
		params.MergedTestChainConfig,
	)

	return pipeline, l1Fetcher
}

// resetAllStages steps the pipeline through all stage resets.
// Requires pipeline.ConfirmEngineReset() to have been called first.
// With Ecotone active at genesis, L1Retrieval.Reset uses BlobDataSource
// which lazily fetches data, so no L1 mock expectations are needed.
func resetAllStages(t *testing.T, pipeline *DerivationPipeline) {
	t.Helper()
	for pipeline.resetting < len(pipeline.stages) {
		_, err := pipeline.Step(context.Background(), eth.L2BlockRef{})
		require.NoError(t, err, "stage reset should succeed")
	}
}

// eventCollector captures CriticalErrorEvent and ResetEvent from the event system.
type eventCollector struct {
	criticalErrors []error
	resetEvents    []error
}

func (c *eventCollector) OnEvent(_ context.Context, ev event.Event) bool {
	switch x := ev.(type) {
	case rollup.CriticalErrorEvent:
		c.criticalErrors = append(c.criticalErrors, x.Err)
		return true
	case rollup.ResetEvent:
		c.resetEvents = append(c.resetEvents, x.Err)
		return true
	}
	return false
}

// TestDepositsOnlyRace_FullPipeline_AfterReset demonstrates the race condition
// using a fully-constructed DerivationPipeline (via NewDerivationPipeline) and
// the real event system.
//
// The scenario: attributes were derived (setting lastAttribs), then a pipeline
// reset clears all stage state including lastAttribs. When a
// DepositsOnlyPayloadAttributesRequestEvent arrives after the reset,
// DepositsOnlyAttributes fails because lastAttribs is nil.
//
// The production event sequence that triggers this:
//  1. Pipeline derives attributes → lastAttribs is set
//  2. BuildStartEvent → onBuildStart → queues ForkchoiceUpdateEvent + BuildStartedEvent
//  3. ForkchoiceUpdateEvent processing triggers a ResetEvent (due to head state change)
//  4. ResetEvent → engine reset → pipeline reset → stages reset → lastAttribs = nil
//  5. BuildStartedEvent → BuildSealEvent → BuildSealedEvent → PayloadProcessEvent
//  6. Payload deemed invalid (Holocene active) → DepositsOnlyPayloadAttributesRequestEvent
//  7. DepositsOnlyAttributes() → lastAttribs == nil → error → CriticalErrorEvent → node crash
func TestDepositsOnlyRace_FullPipeline_AfterReset(t *testing.T) {
	logger := testlog.Logger(t, log.LevelDebug)
	cfg := makeTestRollupConfig()

	// Construct the full derivation pipeline via NewDerivationPipeline.
	// This creates all stages: L1Traversal, L1Retrieval, FrameQueue,
	// ChannelMux, ChannelInReader, BatchMux, AttributesQueue.
	pipeline, _ := newFullPipeline(t, logger, cfg)

	rng := rand.New(rand.NewSource(9999))
	attribs := makeTestAttribs(rng)
	parent := attribs.Parent.ID()
	derivedFrom := attribs.DerivedFrom

	// Step 1: Simulate that the pipeline derived attributes.
	// In production: PipelineStepEvent → pipeline.Step() → NextAttributes() sets lastAttribs.
	pipeline.attrib.lastAttribs = attribs
	require.NotNil(t, pipeline.attrib.lastAttribs)

	// Step 2: Pipeline reset (simulating PipelineDeriver.ResetPipeline() during forceReset).
	pipeline.Reset()
	pipeline.ConfirmEngineReset()

	// Step 3: Step through all stage resets. AttributesQueue.Reset() clears lastAttribs.
	resetAllStages(t, pipeline)
	require.Nil(t, pipeline.attrib.lastAttribs,
		"stage resets should have cleared lastAttribs via AttributesQueue.Reset()")

	// Step 4: Send DepositsOnlyPayloadAttributesRequestEvent through the event system.
	// This simulates what the engine controller does when a payload is deemed invalid
	// under Holocene or denied by the SuperAuthority.
	deriver := NewPipelineDeriver(context.Background(), pipeline)
	ctx := context.Background()
	executor := event.NewGlobalSynchronous(ctx)
	sys := event.NewSystem(logger, executor)
	defer sys.Stop()

	sys.Register("pipeline", deriver)
	collector := &eventCollector{}
	emitter := sys.Register("collector", collector)

	emitter.Emit(ctx, DepositsOnlyPayloadAttributesRequestEvent{
		Parent:      parent,
		DerivedFrom: derivedFrom,
	})
	require.NoError(t, executor.Drain())

	// Verify: PipelineDeriver emits CriticalErrorEvent because lastAttribs
	// was cleared by the pipeline reset.
	require.Len(t, collector.criticalErrors, 1,
		"Expected CriticalErrorEvent when DepositsOnlyPayloadAttributesRequestEvent "+
			"arrives after pipeline reset clears lastAttribs")
	require.Contains(t, collector.criticalErrors[0].Error(), "no attributes generated yet")
	require.Contains(t, collector.criticalErrors[0].Error(), "deriving deposits-only attributes")
}

// TestDepositsOnlyRace_FullPipeline_NeverDerived demonstrates the same error
// path when the pipeline has never derived attributes. This happens when a
// DepositsOnlyPayloadAttributesRequestEvent arrives on a freshly-constructed
// pipeline before any derivation has occurred.
func TestDepositsOnlyRace_FullPipeline_NeverDerived(t *testing.T) {
	logger := testlog.Logger(t, log.LevelDebug)
	cfg := makeTestRollupConfig()

	pipeline, _ := newFullPipeline(t, logger, cfg)

	rng := rand.New(rand.NewSource(7777))
	parent := testutils.RandomL2BlockRef(rng).ID()
	derivedFrom := testutils.RandomBlockRef(rng)

	deriver := NewPipelineDeriver(context.Background(), pipeline)
	ctx := context.Background()
	executor := event.NewGlobalSynchronous(ctx)
	sys := event.NewSystem(logger, executor)
	defer sys.Stop()

	sys.Register("pipeline", deriver)
	collector := &eventCollector{}
	emitter := sys.Register("collector", collector)

	emitter.Emit(ctx, DepositsOnlyPayloadAttributesRequestEvent{
		Parent:      parent,
		DerivedFrom: derivedFrom,
	})
	require.NoError(t, executor.Drain())

	require.Len(t, collector.criticalErrors, 1,
		"Expected CriticalErrorEvent when no attributes have been derived")
	require.Contains(t, collector.criticalErrors[0].Error(), "no attributes generated yet")
	require.Contains(t, collector.criticalErrors[0].Error(), "deriving deposits-only attributes")
}

// TestDepositsOnlyRace_FullPipeline_HappyPath verifies that when lastAttribs
// IS populated (no interleaving reset), the deposits-only flow succeeds and
// emits DerivedAttributesEvent with deposits-only attributes.
func TestDepositsOnlyRace_FullPipeline_HappyPath(t *testing.T) {
	logger := testlog.Logger(t, log.LevelDebug)
	cfg := makeTestRollupConfig()

	pipeline, _ := newFullPipeline(t, logger, cfg)

	// Initialize all pipeline stages by running a reset cycle.
	// FlushChannel (called by DepositsOnlyAttributes) needs
	// BatchMux.SingularBatchProvider initialized during stage reset.
	pipeline.ConfirmEngineReset()
	resetAllStages(t, pipeline)

	rng := rand.New(rand.NewSource(5555))
	attribs := makeTestAttribs(rng)
	parent := attribs.Parent.ID()
	derivedFrom := attribs.DerivedFrom

	// Simulate: pipeline derived attributes after the reset.
	pipeline.attrib.lastAttribs = attribs

	deriver := NewPipelineDeriver(context.Background(), pipeline)
	ctx := context.Background()
	executor := event.NewGlobalSynchronous(ctx)
	sys := event.NewSystem(logger, executor)
	defer sys.Stop()

	sys.Register("pipeline", deriver)

	var gotDerivedAttributes bool
	var derivedAttribs *AttributesWithParent
	collector := event.DeriverFunc(func(_ context.Context, ev event.Event) bool {
		switch x := ev.(type) {
		case DerivedAttributesEvent:
			gotDerivedAttributes = true
			derivedAttribs = x.Attributes
			return true
		case rollup.CriticalErrorEvent:
			t.Fatalf("unexpected CriticalErrorEvent: %v", x.Err)
			return true
		}
		return false
	})
	sys.Register("collector", collector)
	emitter := sys.Register("trigger", nil)

	emitter.Emit(ctx, DepositsOnlyPayloadAttributesRequestEvent{
		Parent:      parent,
		DerivedFrom: derivedFrom,
	})
	require.NoError(t, executor.Drain())

	require.True(t, gotDerivedAttributes,
		"Expected DerivedAttributesEvent with deposits-only attributes")
	require.Equal(t, derivedFrom, derivedAttribs.DerivedFrom)
	require.Equal(t, attribs.Parent, derivedAttribs.Parent)
}

// TestDepositsOnlyRace_PipelineResetClearsLastAttribs verifies the full reset
// mechanism at the pipeline level: Pipeline.Reset() followed by stepping
// through all stage resets clears lastAttribs via AttributesQueue.Reset(),
// making the pipeline unable to serve deposits-only requests.
func TestDepositsOnlyRace_PipelineResetClearsLastAttribs(t *testing.T) {
	logger := testlog.Logger(t, log.LevelDebug)
	cfg := makeTestRollupConfig()

	pipeline, _ := newFullPipeline(t, logger, cfg)

	rng := rand.New(rand.NewSource(3333))
	attribs := makeTestAttribs(rng)
	parent := attribs.Parent.ID()
	derivedFrom := attribs.DerivedFrom

	// First reset cycle to initialize all stages.
	pipeline.ConfirmEngineReset()
	resetAllStages(t, pipeline)

	// Simulate: pipeline derived attributes.
	pipeline.attrib.lastAttribs = attribs

	// Sanity: DepositsOnlyAttributes works when lastAttribs is set.
	result, err := pipeline.DepositsOnlyAttributes(parent, derivedFrom)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Restore lastAttribs (DepositsOnlyAttributes replaces it with deposits-only version).
	pipeline.attrib.lastAttribs = attribs

	// Second reset cycle — this is the one that causes the race.
	pipeline.Reset()
	pipeline.ConfirmEngineReset()
	resetAllStages(t, pipeline)

	require.Nil(t, pipeline.attrib.lastAttribs,
		"AttributesQueue.Reset() should have cleared lastAttribs")

	// DepositsOnlyAttributes now fails.
	_, err = pipeline.DepositsOnlyAttributes(parent, derivedFrom)
	require.Error(t, err)
	require.Contains(t, err.Error(), "no attributes generated yet")
}
