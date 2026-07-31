package driver

import (
	"context"
	gosync "sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/engine"
	"github.com/ethereum-optimism/optimism/op-node/rollup/sequencing"
	"github.com/ethereum-optimism/optimism/op-node/rollup/sync"
	"github.com/ethereum-optimism/optimism/op-service/event"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

// loopTestBlockTime is deliberately huge: the unsafe-gap ticker is derived from it, and
// these tests need the event loop to have no incidental wakeup source at all, so that
// what they measure is only the sequencer wakeup path.
const loopTestBlockTime = 3600

// loopTestTimeout bounds every wait in these tests. The behaviour under test either
// happens in microseconds or never happens, so this is a failure deadline, not a delay.
const loopTestTimeout = 30 * time.Second

// stubSequencer is a SequencerIface whose schedule can be armed and disarmed from a
// foreign goroutine, exactly like the admin_startSequencer/admin_stopSequencer handlers
// do, and which records every schedule read the driver event loop performs.
type stubSequencer struct {
	mu           gosync.Mutex
	nextAction   time.Time
	nextActionOK bool

	signal  chan struct{}
	planned chan bool
}

var _ sequencing.SequencerIface = (*stubSequencer)(nil)

func newStubSequencer() *stubSequencer {
	return &stubSequencer{
		signal:  make(chan struct{}, 1),
		planned: make(chan bool, 128),
	}
}

// setSchedule mimics forceStart/Stop: mutate the schedule and signal, both under the
// same lock that NextAction reads under, so the loop cannot read the old schedule and
// then miss the signal.
func (s *stubSequencer) setSchedule(next time.Time, ok bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextAction, s.nextActionOK = next, ok
	select {
	case s.signal <- struct{}{}:
	default:
	}
}

func (s *stubSequencer) NextAction() (time.Time, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	select {
	case s.planned <- s.nextActionOK:
	default:
	}
	return s.nextAction, s.nextActionOK
}

func (s *stubSequencer) NextActionChanged() <-chan struct{} { return s.signal }

func (s *stubSequencer) OnEvent(context.Context, event.Event) bool   { return false }
func (s *stubSequencer) Init(context.Context, bool) error            { return nil }
func (s *stubSequencer) Start(context.Context, common.Hash) error    { return nil }
func (s *stubSequencer) Stop(context.Context) (common.Hash, error)   { return common.Hash{}, nil }
func (s *stubSequencer) SetMaxSafeLag(context.Context, uint64) error { return nil }
func (s *stubSequencer) OverrideLeader(context.Context) error        { return nil }
func (s *stubSequencer) ConductorEnabled(context.Context) bool       { return false }
func (s *stubSequencer) SetRecoverMode(bool)                         {}
func (s *stubSequencer) Close()                                      {}

func (s *stubSequencer) Active() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.nextActionOK
}

// readyDerivationStub keeps the event loop from resetting the unsafe-gap ticker on
// every iteration; none of its other methods are reachable from these tests.
type readyDerivationStub struct {
	DerivationPipeline
}

func (readyDerivationStub) DerivationReady() bool { return true }

type loopMetricsStub struct {
	Metrics
}

// loopEventRecorder observes the events the driver event loop emits.
type loopEventRecorder struct {
	steps            chan struct{}
	sequencerActions chan time.Time
}

func (r *loopEventRecorder) OnEvent(_ context.Context, ev event.Event) bool {
	switch ev.(type) {
	case StepEvent:
		select {
		case r.steps <- struct{}{}:
		default:
		}
		return true
	case sequencing.SequencerActionEvent:
		select {
		case r.sequencerActions <- time.Now():
		default:
		}
		return true
	}
	return false
}

// startQuiescedEventLoop runs a driver event loop with seq, and returns once the loop
// is parked in its select with no pending work: the only startup activity is the
// initial step request and the StepEvent it produces, both of which are awaited here.
func startQuiescedEventLoop(t *testing.T, seq sequencing.SequencerIface) (*Driver, *loopEventRecorder) {
	logger := testlog.Logger(t, log.LevelDebug)
	ctx, cancel := context.WithCancel(context.Background())

	exec := event.NewGlobalSynchronous(ctx)
	sys := event.NewSystem(logger, exec)
	opts := event.WithNoEmitLimiter()

	sched := NewStepSchedulingDeriver(logger)
	sys.Register("step-scheduler", sched, opts)
	rec := &loopEventRecorder{
		steps:            make(chan struct{}, 8),
		sequencerActions: make(chan time.Time, 8),
	}
	sys.Register("recorder", rec, opts)

	d := &Driver{
		SyncDeriver: &SyncDeriver{
			Derivation:  readyDerivationStub{},
			Engine:      &engine.EngineController{},
			SyncCfg:     &sync.Config{},
			Config:      &rollup.Config{BlockTime: loopTestBlockTime},
			Log:         logger,
			Ctx:         ctx,
			StepDeriver: sched,
		},
		sched:        sched,
		emitter:      sys.Register("driver", nil, opts),
		drain:        exec,
		stateReq:     make(chan chan struct{}),
		forceReset:   make(chan chan struct{}, 10),
		driverConfig: &Config{},
		driverCtx:    ctx,
		driverCancel: cancel,
		log:          logger,
		sequencer:    seq,
		metrics:      &loopMetricsStub{},
	}
	d.wg.Add(1)
	go d.eventLoop()
	t.Cleanup(func() {
		cancel()
		d.wg.Wait()
	})

	// The loop always requests one derivation step on startup. Wait for the resulting
	// StepEvent to be drained, then round-trip a state request to confirm the loop has
	// come back around to its select. After that nothing else can wake it.
	select {
	case <-rec.steps:
	case <-time.After(loopTestTimeout):
		t.Fatal("event loop never emitted its startup step")
	}
	syncWithEventLoop(t, d)
	return d, rec
}

// syncWithEventLoop round-trips a state request, which the loop only services from its
// select. It is a barrier: on return, the loop has completed an iteration.
func syncWithEventLoop(t *testing.T, d *Driver) {
	t.Helper()
	respCh := make(chan struct{})
	select {
	case d.stateReq <- respCh:
	case <-time.After(loopTestTimeout):
		t.Fatal("event loop did not accept a state request")
	}
	select {
	case <-respCh:
	case <-time.After(loopTestTimeout):
		t.Fatal("event loop did not answer a state request")
	}
}

// TestEventLoopWakesWhenSequencerStarts pins the wakeup contract that admin_startSequencer
// depends on: arming the sequencer schedule from an RPC goroutine must make the driver
// loop plan and fire that action, without waiting for an unrelated wakeup.
func TestEventLoopWakesWhenSequencerStarts(t *testing.T) {
	seq := newStubSequencer()
	_, rec := startQuiescedEventLoop(t, seq)

	const armDelay = 100 * time.Millisecond
	start := time.Now()
	seq.setSchedule(start.Add(armDelay), true)

	select {
	case at := <-rec.sequencerActions:
		elapsed := at.Sub(start)
		require.GreaterOrEqual(t, elapsed, armDelay, "must not act before the scheduled time")
		require.Less(t, elapsed, armDelay+time.Second,
			"the loop must observe the newly armed action promptly, not at some later incidental wakeup")
	case <-time.After(loopTestTimeout):
		t.Fatal("driver never acted on the newly armed sequencer action")
	}
}

// TestEventLoopWakesWhenSequencerStops covers the mirror case: disarming the schedule
// must make the loop re-plan and drop its armed timer. Otherwise the timer survives the
// stop and fires a stale sequencer action up to a block-time later.
func TestEventLoopWakesWhenSequencerStops(t *testing.T) {
	seq := newStubSequencer()
	d, rec := startQuiescedEventLoop(t, seq)

	// Arm an action, and barrier so the loop has certainly planned it.
	const armDelay = 250 * time.Millisecond
	seq.setSchedule(time.Now().Add(armDelay), true)
	syncWithEventLoop(t, d)

	// Stopping now, well inside the armed window, must cancel that action outright.
	seq.setSchedule(time.Time{}, false)

	select {
	case <-rec.sequencerActions:
		t.Fatal("driver fired a stale sequencer action after the sequencer was stopped")
	case <-time.After(armDelay * 6):
	}
}

// TestEventLoopStaysQuietWithoutSchedule guards against the wakeup turning into a busy
// loop: with nothing armed and nothing signalled, the loop must not spin.
func TestEventLoopStaysQuietWithoutSchedule(t *testing.T) {
	seq := newStubSequencer()
	d, _ := startQuiescedEventLoop(t, seq)

	// Drain the plans recorded during startup, then take two barriers. Each barrier
	// costs exactly one iteration, so a quiet loop yields at most one plan per barrier
	// plus one that may already have been in flight.
	for len(seq.planned) > 0 {
		<-seq.planned
	}
	syncWithEventLoop(t, d)
	syncWithEventLoop(t, d)

	require.LessOrEqual(t, len(seq.planned), 3,
		"event loop iterated more than the barriers required: it is spinning")
}

// TestEventLoopSignalDuringPlanIsNotLost covers the interleaving the buffered signal
// exists for: a schedule change that lands while the loop is between planning and
// blocking in its select must still be observed.
func TestEventLoopSignalDuringPlanIsNotLost(t *testing.T) {
	seq := newStubSequencer()
	_, rec := startQuiescedEventLoop(t, seq)

	// Churn the schedule so a change lands at arbitrary points of the loop iteration,
	// including after NextAction was read but before the loop reaches its select. The
	// final state is armed, so the loop owes us exactly one sequencer action.
	for range 200 {
		seq.setSchedule(time.Now(), true)
		seq.setSchedule(time.Time{}, false)
	}
	seq.setSchedule(time.Now(), true)

	select {
	case <-rec.sequencerActions:
	case <-time.After(loopTestTimeout):
		t.Fatal("driver lost the sequencer wakeup")
	}
}
