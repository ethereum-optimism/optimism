package sequencing

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math/rand" // nosemgrep
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/metrics"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-node/rollup/engine"
	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/event"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
)

// ChaoticEngine simulates what the Engine deriver would do, upon events from the sequencer.
// But does so with repeated errors and bad time delays.
// It is up to the sequencer code to recover from the errors and keep the
// onchain time accurate to the simulated offchain time.
type ChaoticEngine struct {
	t *testing.T

	rng *rand.Rand

	emitter event.Emitter

	clock interface {
		Now() time.Time
		Set(t time.Time)
	}

	currentPayloadInfo eth.PayloadInfo
	currentAttributes  *derive.AttributesWithParent

	unsafe, safe, finalized eth.L2BlockRef
}

func (c *ChaoticEngine) clockRandomIncrement(minIncr, maxIncr time.Duration) {
	require.LessOrEqual(c.t, minIncr, maxIncr, "sanity check time duration range")
	incr := minIncr + time.Duration(c.rng.Int63n(int64(maxIncr-minIncr)))
	c.clock.Set(c.clock.Now().Add(incr))
}

// RequestForkchoiceUpdate implements SequencerEngine.
func (c *ChaoticEngine) RequestForkchoiceUpdate(ctx context.Context) {
	c.emitter.Emit(ctx, engine.ForkchoiceUpdateEvent{
		UnsafeL2Head:    c.unsafe,
		SafeL2Head:      c.safe,
		FinalizedL2Head: c.finalized,
	})
}

// StartBuild implements SequencerEngine.
func (c *ChaoticEngine) StartBuild(ctx context.Context, attrs *derive.AttributesWithParent) (*engine.BuildStartResult, error) {
	c.currentPayloadInfo = eth.PayloadInfo{}
	_, err := c.rng.Read(c.currentPayloadInfo.ID[:])
	require.NoError(c.t, err)
	c.currentPayloadInfo.Timestamp = uint64(attrs.Attributes.Timestamp)

	c.clockRandomIncrement(0, time.Millisecond*300)
	if c.rng.Intn(10) == 0 {
		c.clockRandomIncrement(0, time.Second*2)
	}

	p := c.rng.Float32()
	switch {
	case p < 0.05:
		c.emitter.Emit(ctx, engine.BuildInvalidEvent{
			Attributes: attrs,
			Err:        errors.New("mock start invalid error"),
		})
		return nil, errors.New("mock start invalid error")
	case p < 0.07:
		c.emitter.Emit(ctx, rollup.ResetEvent{
			Err: errors.New("mock reset on start error"),
		})
		return nil, errors.New("mock reset on start error")
	case p < 0.12:
		c.emitter.Emit(ctx, rollup.EngineTemporaryErrorEvent{
			Err: errors.New("mock temp start error"),
		})
		return nil, errors.New("mock temp start error")
	default:
		c.currentAttributes = attrs
		return &engine.BuildStartResult{
			Info:         c.currentPayloadInfo,
			BuildStarted: c.clock.Now(),
			Parent:       attrs.Parent,
		}, nil
	}
}

// SealBuild implements SequencerEngine.
func (c *ChaoticEngine) SealBuild(ctx context.Context, info eth.PayloadInfo, buildStarted time.Time) (*engine.SealResult, error) {
	c.clockRandomIncrement(0, time.Millisecond*300)

	if c.currentPayloadInfo == (eth.PayloadInfo{}) {
		c.emitter.Emit(ctx, engine.PayloadSealExpiredErrorEvent{
			Info:        info,
			Err:         errors.New("job was cancelled"),
			Concluding:  false,
			DerivedFrom: eth.L1BlockRef{},
		})
		return nil, errors.New("job was cancelled")
	}
	require.Equal(c.t, c.currentPayloadInfo, info, "seal the current payload")
	require.NotNil(c.t, c.currentAttributes, "must have started building")

	if c.rng.Intn(20) == 0 {
		c.clockRandomIncrement(0, time.Second*3)
	}

	p := c.rng.Float32()
	switch {
	case p < 0.03:
		c.emitter.Emit(ctx, engine.PayloadSealInvalidEvent{
			Info:        info,
			Err:         errors.New("mock invalid seal"),
			Concluding:  false,
			DerivedFrom: eth.L1BlockRef{},
		})
		c.currentPayloadInfo = eth.PayloadInfo{}
		c.currentAttributes = nil
		return nil, errors.New("mock invalid seal")
	case p < 0.08:
		c.emitter.Emit(ctx, engine.PayloadSealExpiredErrorEvent{
			Info:        info,
			Err:         errors.New("mock temp engine error"),
			Concluding:  false,
			DerivedFrom: eth.L1BlockRef{},
		})
		c.currentPayloadInfo = eth.PayloadInfo{}
		c.currentAttributes = nil
		return nil, errors.New("mock temp engine error")
	default:
		payloadEnvelope := &eth.ExecutionPayloadEnvelope{
			ParentBeaconBlockRoot: c.currentAttributes.Attributes.ParentBeaconBlockRoot,
			ExecutionPayload: &eth.ExecutionPayload{
				ParentHash:   c.currentAttributes.Parent.Hash,
				FeeRecipient: c.currentAttributes.Attributes.SuggestedFeeRecipient,
				BlockNumber:  eth.Uint64Quantity(c.currentAttributes.Parent.Number + 1),
				BlockHash:    testutils.RandomHash(c.rng),
				Timestamp:    c.currentAttributes.Attributes.Timestamp,
				Transactions: c.currentAttributes.Attributes.Transactions,
			},
		}
		l1Origin := decodeID(c.currentAttributes.Attributes.Transactions[0])
		payloadRef := eth.L2BlockRef{
			Hash:           payloadEnvelope.ExecutionPayload.BlockHash,
			Number:         uint64(payloadEnvelope.ExecutionPayload.BlockNumber),
			ParentHash:     payloadEnvelope.ExecutionPayload.ParentHash,
			Time:           uint64(payloadEnvelope.ExecutionPayload.Timestamp),
			L1Origin:       l1Origin,
			SequenceNumber: 0,
		}
		c.currentPayloadInfo = eth.PayloadInfo{}
		c.currentAttributes = nil
		return &engine.SealResult{
			Envelope: payloadEnvelope,
			Ref:      payloadRef,
		}, nil
	}
}

// ProcessPayload implements SequencerEngine.
func (c *ChaoticEngine) ProcessPayload(ctx context.Context, envelope *eth.ExecutionPayloadEnvelope, ref eth.L2BlockRef, buildStarted time.Time) error {
	c.clockRandomIncrement(0, time.Millisecond*500)

	p := c.rng.Float32()
	switch {
	case p < 0.05:
		c.emitter.Emit(ctx, rollup.EngineTemporaryErrorEvent{
			Err: errors.New("mock temp engine error"),
		})
		return errors.New("mock temp engine error")
	case p < 0.08:
		c.emitter.Emit(ctx, engine.PayloadInvalidEvent{
			Envelope: envelope,
			Err:      errors.New("mock invalid payload"),
		})
		return engine.ErrPayloadInvalid
	default:
		if p < 0.13 {
			c.clockRandomIncrement(0, time.Second*3)
		}
		c.unsafe = ref
		// Emit UnsafeUpdateEvent and ForkchoiceUpdateEvent to simulate what the real engine does
		c.emitter.Emit(ctx, engine.UnsafeUpdateEvent{Ref: ref})
		c.emitter.Emit(ctx, engine.ForkchoiceUpdateEvent{
			UnsafeL2Head:    c.unsafe,
			SafeL2Head:      c.safe,
			FinalizedL2Head: c.finalized,
		})
		return nil
	}
}

// OnEvent handles events that are still routed via the event system (resets, errors, etc.)
func (c *ChaoticEngine) OnEvent(ctx context.Context, ev event.Event) bool {
	switch ev := ev.(type) {
	case rollup.EngineTemporaryErrorEvent:
		c.clockRandomIncrement(0, time.Millisecond*100)
		c.currentPayloadInfo = eth.PayloadInfo{}
		c.currentAttributes = nil
	case rollup.ResetEvent:
		c.clockRandomIncrement(0, time.Second*4)
		c.currentPayloadInfo = eth.PayloadInfo{}
		c.currentAttributes = nil
		c.emitter.Emit(ctx, engine.EngineResetConfirmedEvent{
			LocalUnsafe: c.unsafe,
			CrossUnsafe: c.unsafe,
			LocalSafe:   c.safe,
			CrossSafe:   c.safe,
			Finalized:   c.finalized,
		})
	case engine.BuildInvalidEvent:
		c.clockRandomIncrement(0, time.Millisecond*50)
		c.currentPayloadInfo = eth.PayloadInfo{}
		c.currentAttributes = nil
		c.emitter.Emit(ctx, engine.InvalidPayloadAttributesEvent(ev))
	case engine.BuildCancelEvent:
		c.currentPayloadInfo = eth.PayloadInfo{}
		c.currentAttributes = nil
	default:
		return false
	}
	return true
}

func (c *ChaoticEngine) AttachEmitter(em event.Emitter) {
	c.emitter = em
}

var _ event.Deriver = (*ChaoticEngine)(nil)
var _ SequencerEngine = (*ChaoticEngine)(nil)

// TestSequencerChaos runs the sequencer with a simulated engine,
// mocking different kinds of errors and timing issues.
func TestSequencerChaos(t *testing.T) {
	for i := int64(1); i < 100; i++ {
		t.Run(fmt.Sprintf("simulation-%d", i), func(t *testing.T) {
			testSequencerChaosWithSeed(t, i)
		})
	}
}

func testSequencerChaosWithSeed(t *testing.T, seed int64) {
	logger := testlog.Logger(t, log.LevelCrit)

	rng := rand.New(rand.NewSource(seed))

	testClock := clock.NewSimpleClock()

	eng := &ChaoticEngine{
		t:     t,
		rng:   rng,
		clock: testClock,
	}

	// Create sequencer with the chaotic engine as the SequencerEngine
	cfg := &rollup.Config{
		Genesis: rollup.Genesis{
			L1: eth.BlockID{
				Hash:   testutils.RandomHash(rng),
				Number: 3000000,
			},
			L2: eth.BlockID{
				Hash:   testutils.RandomHash(rng),
				Number: 0,
			},
			L2Time:       10000000,
			SystemConfig: eth.SystemConfig{},
		},
		BlockTime:         2,
		MaxSequencerDrift: 15 * 60,
		RegolithTime:      new(uint64),
		CanyonTime:        new(uint64),
		DeltaTime:         new(uint64),
		EcotoneTime:       new(uint64),
		FjordTime:         new(uint64),
		GraniteTime:       new(uint64),
		HoloceneTime:      new(uint64),
		IsthmusTime:       new(uint64),
		JovianTime:        new(uint64),
	}

	testClock.SetTime(cfg.Genesis.L2Time)

	attribBuilder := &FakeAttributesBuilder{cfg: cfg, rng: rng}
	seqState := &BasicSequencerStateListener{}
	cond := &FakeConductor{leader: true}
	asyncGossip := &FakeAsyncGossip{}

	genesisRef := eth.L2BlockRef{
		Hash:           cfg.Genesis.L2.Hash,
		Number:         cfg.Genesis.L2.Number,
		ParentHash:     common.Hash{},
		Time:           cfg.Genesis.L2Time,
		L1Origin:       cfg.Genesis.L1,
		SequenceNumber: 0,
	}
	eng.finalized = genesisRef
	eng.safe = genesisRef
	eng.unsafe = genesisRef

	var l1OriginSelectErr error
	l1BlockHash := func(num uint64) (out common.Hash) {
		out[0] = 1
		binary.BigEndian.PutUint64(out[32-8:], num)
		return
	}
	l1OriginSelector := &FakeL1OriginSelector{
		l1OriginFn: func(l2Head eth.L2BlockRef) (eth.L1BlockRef, error) {
			if l1OriginSelectErr != nil {
				return eth.L1BlockRef{}, l1OriginSelectErr
			}
			if l2Head.Number == genesisRef.Number {
				return eth.L1BlockRef{
					Hash:       genesisRef.L1Origin.Hash,
					Number:     genesisRef.L1Origin.Number,
					Time:       genesisRef.Time,
					ParentHash: common.Hash{},
				}, nil
			}
			origin := eth.L1BlockRef{
				Hash:       l2Head.L1Origin.Hash,
				Number:     l2Head.L1Origin.Number,
				ParentHash: l1BlockHash(l2Head.L1Origin.Number - 1),
				Time:       genesisRef.Time + (l2Head.L1Origin.Number-genesisRef.L1Origin.Number)*12,
			}
			if l2Head.Time+cfg.BlockTime > origin.Time+cfg.MaxSequencerDrift {
				origin.Number += 1
				origin.ParentHash = origin.Hash
				origin.Hash = l1BlockHash(origin.Number)
				origin.Time += 12
			}
			return origin, nil
		},
	}

	seq := NewSequencer(context.Background(), logger, cfg, defaultSealingDuration, attribBuilder,
		l1OriginSelector, seqState, cond,
		asyncGossip, metrics.NoopMetrics, eng)
	seq.timeNow = testClock.Now
	seq.toBlockRef = func(rollupCfg *rollup.Config, payload *eth.ExecutionPayload) (eth.L2BlockRef, error) {
		return eth.L2BlockRef{
			Hash:           payload.BlockHash,
			Number:         uint64(payload.BlockNumber),
			ParentHash:     payload.ParentHash,
			Time:           uint64(payload.Timestamp),
			L1Origin:       decodeID(payload.Transactions[0]),
			SequenceNumber: 0,
		}, nil
	}

	ex := event.NewGlobalSynchronous(context.Background())
	sys := event.NewSystem(logger, ex)
	sys.AddTracer(event.NewLogTracer(logger, log.LevelInfo))
	opts := event.WithNoEmitLimiter()
	sys.Register("sequencer", seq, opts)
	// Register the chaotic engine as an event deriver too (for reset/error handling)
	sys.Register("engine", eng, opts)
	testEm := sys.Register("test", nil, opts)

	// Init sequencer, as active
	require.NoError(t, seq.Init(context.Background(), true))
	require.NoError(t, ex.Drain(), "initial forkchoice update etc. completes")

	testEm.Emit(context.Background(), engine.ForkchoiceUpdateEvent{
		UnsafeL2Head:    genesisRef,
		SafeL2Head:      genesisRef,
		FinalizedL2Head: genesisRef,
	})

	genesisTime := time.Unix(int64(cfg.Genesis.L2Time), 0)

	i := 0
	sanityCap := 1000
	targetBlocks := uint64(100)
	for eng.unsafe.Number < cfg.Genesis.L2.Number+targetBlocks && i < sanityCap {
		simPast := eng.clock.Now().Sub(genesisTime)
		onchainPast := time.Unix(int64(eng.unsafe.Time), 0).Sub(genesisTime)
		logger.Info("Simulation step", "i", i, "sim_time", simPast,
			"onchain_time", onchainPast,
			"relative", simPast-onchainPast, "blocks", eng.unsafe.Number-cfg.Genesis.L2.Number)

		eng.clockRandomIncrement(0, time.Millisecond*10)

		drainErr := ex.DrainUntil(func(ev event.Event) bool {
			return rng.Intn(10) == 0
		}, false)

		nextTime, ok := seq.NextAction()
		if drainErr == io.EOF && !ok {
			t.Fatalf("No action scheduled, but also no events to change inputs left")
		}
		if ok && testClock.Now().After(nextTime) {
			seq.RunAction(context.Background())
		} else {
			waitTime := nextTime.Sub(eng.clock.Now())
			if drainErr == io.EOF {
				logger.Info("No events left, skipping forward to next sequencing action", "wait", waitTime)
				eng.clockRandomIncrement(waitTime, waitTime+time.Millisecond*10)
			} else {
				logger.Info("Not sequencing time yet, processing more events first", "wait", waitTime)
			}
		}

		i += 1
	}

	blocksSinceGenesis := eng.unsafe.Number - cfg.Genesis.L2.Number
	if i >= sanityCap {
		t.Fatalf("Sequenced %d blocks, ran out of simulation steps", blocksSinceGenesis)
	}
	require.Equal(t, targetBlocks, blocksSinceGenesis)

	now := testClock.Now()
	timeSinceGenesis := now.Sub(genesisTime)
	idealTimeSinceGenesis := time.Duration(blocksSinceGenesis*cfg.BlockTime) * time.Second
	diff := timeSinceGenesis - idealTimeSinceGenesis
	if diff.Abs() > time.Second*35 {
		t.Fatalf("Failed to maintain target time. Spent %s, but target was %s",
			timeSinceGenesis, idealTimeSinceGenesis)
	}
}
