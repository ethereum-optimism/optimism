package sequencing

import (
	"context"
	"log/slog"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-node/rollup/engine"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/event"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

// logSignalHandler signals a channel whenever a log record containing match
// passes through. Safe to select on from the test goroutine while the loop
// goroutine logs.
type logSignalHandler struct {
	inner slog.Handler
	match string
	ch    chan struct{}
}

func (s *logSignalHandler) Enabled(ctx context.Context, lvl slog.Level) bool {
	return s.inner.Enabled(ctx, lvl)
}

func (s *logSignalHandler) Handle(ctx context.Context, r slog.Record) error {
	if strings.Contains(r.Message, s.match) {
		select {
		case s.ch <- struct{}{}:
		default:
		}
	}
	return s.inner.Handle(ctx, r)
}

func (s *logSignalHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &logSignalHandler{inner: s.inner.WithAttrs(attrs), match: s.match, ch: s.ch}
}

func (s *logSignalHandler) WithGroup(name string) slog.Handler {
	return &logSignalHandler{inner: s.inner.WithGroup(name), match: s.match, ch: s.ch}
}

// runLoopHarness runs the real RunLoop against an active sequencer with a
// future deadline armed, exactly the state scheduleNextAction leaves behind
// after a block insertion. The clock offset simulates the wall clock being
// stepped relative to the monotonic clock (as VM time-sync daemons do): the
// sequencer's clock reads time.Now().Add(offset), while the loop's timer
// sleeps monotonic time.
type runLoopHarness struct {
	seq          *Sequencer
	offsetNs     atomic.Int64
	buildStarted chan struct{}
	earlyFire    chan struct{}
	head         eth.L2BlockRef
	deadline     time.Time
}

func newRunLoopHarness(t *testing.T) *runLoopHarness {
	h := &runLoopHarness{
		buildStarted: make(chan struct{}, 16),
		earlyFire:    make(chan struct{}, 1),
	}
	logger := testlog.LoggerWithHandlerMod(t, log.LevelDebug, func(inner slog.Handler) slog.Handler {
		return &logSignalHandler{inner: inner, match: "fired before its wall-clock deadline", ch: h.earlyFire}
	})
	seq, deps := createSequencer(logger)
	// The harness asserts no emitted events; a permissive emitter keeps an
	// accidental Emit from panicking inside the loop goroutine.
	seq.AttachEmitter(event.NoopEmitter{})
	deps.conductor.leader = true

	h.seq = seq
	seq.timeNow = func() time.Time { return time.Now().Add(time.Duration(h.offsetNs.Load())) }

	h.head = eth.L2BlockRef{Hash: common.Hash{0xaa}, Number: 100, Time: uint64(time.Now().Unix())}
	deps.l1OriginSelector.l1OriginFn = func(l2Head eth.L2BlockRef) (eth.L1BlockRef, error) {
		return eth.L1BlockRef{Number: 3000001, Time: h.head.Time}, nil
	}
	deps.eng.startBuildFn = func(ctx context.Context, attrs *derive.AttributesWithParent) (*engine.BuildStartResult, error) {
		h.buildStarted <- struct{}{}
		// Drop the job; the sequencer parks afterwards, which keeps the
		// harness simple: these tests only care whether the action fired.
		return nil, engine.ErrStaleBuild
	}

	ctx := context.Background()
	require.NoError(t, seq.Init(ctx, false))
	deliver(seq, engine.ForkchoiceUpdateEvent{UnsafeL2Head: h.head})
	seq.latestSealed = h.head
	require.NoError(t, seq.Start(ctx, h.head.Hash))

	// Overwrite the schedule with a future deadline, as scheduleNextAction
	// leaves it right after a block insertion.
	h.deadline = time.Now().Add(500 * time.Millisecond)
	seq.l.Lock()
	seq.nextAction = h.deadline
	seq.nextActionOK = true
	seq.l.Unlock()
	return h
}

// start runs RunLoop until the test ends.
func (h *runLoopHarness) start(t *testing.T) {
	loopCtx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		h.seq.RunLoop(loopCtx)
	}()
	t.Cleanup(func() {
		cancel()
		<-done
	})
}

// TestRunLoopActionFiresOnSchedule is the control for the early-fire test
// below: with a well-behaved clock, the armed deadline fires and a build
// starts.
func TestRunLoopActionFiresOnSchedule(t *testing.T) {
	h := newRunLoopHarness(t)
	h.start(t)
	select {
	case <-h.buildStarted:
	case <-time.After(10 * time.Second):
		t.Fatal("armed deadline never fired under a healthy clock")
	}
}

// TestRunLoopSurvivesEarlyTimerFire covers the deadline scheduling against
// wall-clock regressions. The armed deadlines are wall-only time.Unix values,
// so the timer duration is a wall-clock difference while the timer itself
// sleeps monotonic time: if the wall clock is stepped backward between arm
// and fire (VM time sync, NTP), the timer delivers while the sequencer clock
// still reads before the deadline and RunLoop takes the not-yet-due continue
// path.
//
// RunLoop used to record lastRan = armed before that check. The planning pass
// refuses to arm a deadline equal to lastRan (the anti-spin guard), and
// lastRan was only reset when the schedule emptied — which never happens
// while a valid, overdue action is scheduled. One early delivery therefore
// lost the deadline forever: the sequencer sat parked in its select with
// admin_sequencerActive=true and block production permanently frozen, with no
// log, metric, or error. Observed as intermittent total L2 freezes on Docker
// Desktop devnets, where the VM wall clock is periodically stepped by up to
// ~10ms.
func TestRunLoopSurvivesEarlyTimerFire(t *testing.T) {
	h := newRunLoopHarness(t)
	h.start(t)

	// Let the loop consume the pending wake-up from Start() and settle into
	// its timer wait, so the deadline's duration is computed pre-regression.
	time.Sleep(50 * time.Millisecond)

	// Step the wall clock 1s backward while the timer (armed ~450ms out) is
	// sleeping: the monotonic timer will deliver while the sequencer's clock
	// still reads before the deadline.
	h.offsetNs.Store(int64(-time.Second))

	// Wait until the loop demonstrably handles the early fire. On the pre-fix
	// code the log never appears: the deadline is silently lost instead.
	select {
	case <-h.buildStarted:
		// The regression landed after the timer had already fired (heavily
		// loaded machine); the scenario under test did not occur.
		t.Skip("timer fired before the clock regression took effect")
	case <-h.earlyFire:
	case <-time.After(10 * time.Second):
		t.Fatal("RunLoop never handled the early timer fire: pre-fix code silently drops the deadline here")
	}

	// While the clock still reads before the deadline, wake-ups must re-plan
	// without running the action early — and without losing the deadline.
	h.seq.OnEvent(context.Background(), engine.ForkchoiceUpdateEvent{UnsafeL2Head: h.head})
	time.Sleep(50 * time.Millisecond)
	select {
	case <-h.buildStarted:
		t.Fatal("action ran while the sequencer clock still read before its deadline")
	default:
	}

	// The regression was transient: once the clock recovers, the re-armed
	// deadline must fire and the action must run.
	h.offsetNs.Store(0)
	select {
	case <-h.buildStarted:
	case <-time.After(10 * time.Second):
		next, ok := h.seq.NextAction()
		t.Fatalf("deadline lost to the early timer fire (NextAction still claims ok=%v at %s, %s overdue)",
			ok, next, time.Since(next))
	}
}
