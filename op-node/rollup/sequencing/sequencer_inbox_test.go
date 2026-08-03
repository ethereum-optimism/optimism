package sequencing

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-node/rollup/engine"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
)

// TestSequencerInbox_ResetConfirmOrdering checks that inbox replay preserves
// arrival order for the reset/confirm pair: a confirmation after a reset
// resumes the sequencer, while a reset after a confirmation keeps it paused.
func TestSequencerInbox_ResetConfirmOrdering(t *testing.T) {
	ctx := context.Background()

	t.Run("reset then confirm resumes", func(t *testing.T) {
		s := newSeqSetup(t)
		s.seq.OnEvent(ctx, rollup.ResetEvent{Err: errors.New("mock reset")})
		s.seq.OnEvent(ctx, engine.EngineResetConfirmedEvent{})
		s.seq.drainInbox()
		next, ok := s.seq.NextAction()
		require.True(t, ok, "confirmed reset resumes sequencing")
		require.Equal(t, s.clock.Now().Add(s.blockTime()), next,
			"resume after one block of engine cool-down")
	})

	t.Run("confirm then reset stays paused", func(t *testing.T) {
		s := newSeqSetup(t)
		s.seq.OnEvent(ctx, engine.EngineResetConfirmedEvent{})
		s.seq.OnEvent(ctx, rollup.ResetEvent{Err: errors.New("mock reset")})
		s.seq.drainInbox()
		_, ok := s.seq.NextAction()
		require.False(t, ok, "unconfirmed reset keeps the sequencer paused")
	})
}

// TestSequencerInbox_PauseHeadUpdateOrdering checks arrival-order replay of the
// derivation-is-building pause against head updates: a head update after the
// pause un-pauses the sequencer, while the reverse order stays paused.
func TestSequencerInbox_PauseHeadUpdateOrdering(t *testing.T) {
	ctx := context.Background()
	derivedBuild := func(s *seqTestSetup) engine.BuildStartedEvent {
		return engine.BuildStartedEvent{
			Info:        eth.PayloadInfo{ID: eth.PayloadID{0x77}, Timestamp: s.head.Time + s.deps.cfg.BlockTime},
			Parent:      s.head,
			DerivedFrom: eth.L1BlockRef{Hash: common.Hash{0x01}, Number: 1},
		}
	}
	newerHead := func(s *seqTestSetup) engine.ForkchoiceUpdateEvent {
		return engine.ForkchoiceUpdateEvent{UnsafeL2Head: eth.L2BlockRef{
			Hash:     common.Hash{0x23},
			Number:   s.head.Number + 1,
			L1Origin: s.head.L1Origin,
			Time:     s.head.Time + s.deps.cfg.BlockTime,
		}}
	}

	t.Run("pause then head update resumes", func(t *testing.T) {
		s := newSeqSetup(t)
		s.seq.OnEvent(ctx, derivedBuild(s))
		s.seq.OnEvent(ctx, newerHead(s))
		s.seq.drainInbox()
		_, ok := s.seq.NextAction()
		require.True(t, ok, "head update after derivation-build pause re-arms the sequencer")
	})

	t.Run("head update then pause stays paused", func(t *testing.T) {
		s := newSeqSetup(t)
		s.seq.OnEvent(ctx, newerHead(s))
		s.seq.OnEvent(ctx, derivedBuild(s))
		s.seq.drainInbox()
		_, ok := s.seq.NextAction()
		require.False(t, ok, "derivation-build pause after head update keeps the sequencer paused")
	})

	t.Run("sequencer-origin build-start is ignored", func(t *testing.T) {
		s := newSeqSetup(t)
		nextBefore, okBefore := s.seq.NextAction()
		ev := derivedBuild(s)
		ev.DerivedFrom = eth.L1BlockRef{} // sequencer builds are handled inline via direct calls
		s.seq.OnEvent(ctx, ev)
		s.seq.drainInbox()
		next, ok := s.seq.NextAction()
		require.Equal(t, okBefore, ok, "schedule unchanged")
		require.Equal(t, nextBefore, next, "schedule unchanged")
	})
}

// TestSequencerInbox_CoalescesAdjacentHeadUpdates checks the single inbox
// coalescing rule: a head update replaces the last inbox entry iff that entry
// is also a head update. Entries of other kinds are never reordered or merged.
func TestSequencerInbox_CoalescesAdjacentHeadUpdates(t *testing.T) {
	ctx := context.Background()
	logger := testlog.Logger(t, log.LevelError)
	refA := eth.L2BlockRef{Hash: common.Hash{0xa1}, Number: 1}
	refB := eth.L2BlockRef{Hash: common.Hash{0xb2}, Number: 2}
	refC := eth.L2BlockRef{Hash: common.Hash{0xc3}, Number: 3}

	t.Run("adjacent head updates collapse to the latest", func(t *testing.T) {
		seq, _ := createSequencer(logger)
		seq.OnEvent(ctx, engine.ForkchoiceUpdateEvent{UnsafeL2Head: refA})
		seq.OnEvent(ctx, engine.ForkchoiceUpdateEvent{UnsafeL2Head: refB})
		require.Len(t, seq.inbox, 1, "adjacent head updates coalesce")
		fcu, isFCU := seq.inbox[0].(engine.ForkchoiceUpdateEvent)
		require.True(t, isFCU)
		require.Equal(t, refB, fcu.UnsafeL2Head, "latest head wins")
	})

	t.Run("no coalescing across other kinds", func(t *testing.T) {
		seq, _ := createSequencer(logger)
		seq.OnEvent(ctx, engine.ForkchoiceUpdateEvent{UnsafeL2Head: refA})
		seq.OnEvent(ctx, rollup.ResetEvent{Err: errors.New("mock reset")})
		seq.OnEvent(ctx, engine.ForkchoiceUpdateEvent{UnsafeL2Head: refC})
		require.Len(t, seq.inbox, 3, "reset entry blocks coalescing")
		first, isFCU := seq.inbox[0].(engine.ForkchoiceUpdateEvent)
		require.True(t, isFCU)
		require.Equal(t, refA, first.UnsafeL2Head, "pre-reset head is preserved")
		require.IsType(t, rollup.ResetEvent{}, seq.inbox[1], "arrival order is preserved")
		last, isFCU := seq.inbox[2].(engine.ForkchoiceUpdateEvent)
		require.True(t, isFCU)
		require.Equal(t, refC, last.UnsafeL2Head)
	})

	t.Run("forkchoice-init events coalesce as head updates", func(t *testing.T) {
		seq, _ := createSequencer(logger)
		seq.OnEvent(ctx, engine.ForkchoiceUpdateEvent{UnsafeL2Head: refA})
		seq.OnEvent(ctx, engine.ForkchoiceUpdateInitEvent{UnsafeL2Head: refB})
		require.Len(t, seq.inbox, 1, "init event is ingested as a head update and coalesces")
		fcu, isFCU := seq.inbox[0].(engine.ForkchoiceUpdateEvent)
		require.True(t, isFCU)
		require.Equal(t, refB, fcu.UnsafeL2Head)
	})

	t.Run("rewinding head update does not coalesce", func(t *testing.T) {
		// Head updates are not monotonic across block replacements or
		// backup-unsafe restores. Coalescing [FCU(3), FCU(1)] to FCU(1) would
		// skip the drop-stale-job check replay applies at head 3, letting a
		// build job onto a reorged-out block survive.
		seq, _ := createSequencer(logger)
		seq.OnEvent(ctx, engine.ForkchoiceUpdateEvent{UnsafeL2Head: refC})
		seq.OnEvent(ctx, engine.ForkchoiceUpdateEvent{UnsafeL2Head: refA})
		require.Len(t, seq.inbox, 2, "rewinding head update is appended, not coalesced")
		first, isFCU := seq.inbox[0].(engine.ForkchoiceUpdateEvent)
		require.True(t, isFCU)
		require.Equal(t, refC, first.UnsafeL2Head, "higher head is preserved for replay")
		last, isFCU := seq.inbox[1].(engine.ForkchoiceUpdateEvent)
		require.True(t, isFCU)
		require.Equal(t, refA, last.UnsafeL2Head)
	})
}

// TestSequencerInbox_WakeCoalescing checks the cap-1 wake channel: any number
// of inbox appends leaves at most one pending wake, non-ingest events neither
// enqueue nor wake, and a consumed wake is re-armed by the next append.
func TestSequencerInbox_WakeCoalescing(t *testing.T) {
	ctx := context.Background()
	logger := testlog.Logger(t, log.LevelError)
	seq, _ := createSequencer(logger)
	ref := eth.L2BlockRef{Hash: common.Hash{0xa1}, Number: 1}

	require.Empty(t, seq.wakeCh, "no wake pending initially")

	require.True(t, seq.OnEvent(ctx, engine.ForkchoiceUpdateEvent{UnsafeL2Head: ref}))
	require.Len(t, seq.wakeCh, 1, "append wakes the sequencer")

	require.True(t, seq.OnEvent(ctx, rollup.ResetEvent{Err: errors.New("mock reset")}))
	require.True(t, seq.OnEvent(ctx, engine.ForkchoiceUpdateEvent{UnsafeL2Head: ref}))
	require.Len(t, seq.wakeCh, 1, "wakes coalesce; no busy-looping wake storm")

	require.False(t, seq.OnEvent(ctx, engine.BuildSealedEvent{}), "not an ingest kind")
	require.Len(t, seq.inbox, 3, "non-ingest events are not enqueued")
	require.Len(t, seq.wakeCh, 1, "non-ingest events do not wake")

	<-seq.wakeCh
	require.Empty(t, seq.wakeCh)
	seq.drainInbox()
	require.Empty(t, seq.inbox, "drain takes the whole log")

	require.True(t, seq.OnEvent(ctx, engine.ForkchoiceUpdateEvent{UnsafeL2Head: ref}))
	require.Len(t, seq.wakeCh, 1, "consumed wake is re-armed by the next append")
}

// TestSequencerWake_StartStop checks that the RPC-driven lifecycle transitions
// fire the wake channel, so a parked run-loop re-plans its schedule promptly.
func TestSequencerWake_StartStop(t *testing.T) {
	logger := testlog.Logger(t, log.LevelError)
	seq, deps := createSequencer(logger)
	seq.AttachEmitter(&testutils.MockEmitter{})
	deps.conductor.leader = true

	require.NoError(t, seq.Init(context.Background(), false))

	head := eth.L2BlockRef{Hash: common.Hash{0xaa}}
	deliver(seq, engine.ForkchoiceUpdateEvent{UnsafeL2Head: head})
	seq.latestSealed = head
	// consume the ingest wake, to isolate the lifecycle wakes below
	select {
	case <-seq.wakeCh:
	default:
	}

	require.NoError(t, seq.Start(context.Background(), head.Hash))
	require.Len(t, seq.wakeCh, 1, "start wakes the run-loop")
	<-seq.wakeCh

	_, err := seq.Stop(context.Background())
	require.NoError(t, err)
	require.Len(t, seq.wakeCh, 1, "stop wakes the run-loop to disarm its timer")
}

// TestSequencerInbox_OnEventNonBlocking checks that event ingestion does not
// take the sequencer action lock: the event loop must never stall behind a
// long-running sequencer action (attribute preparation, conductor commit, etc).
func TestSequencerInbox_OnEventNonBlocking(t *testing.T) {
	logger := testlog.Logger(t, log.LevelError)
	seq, _ := createSequencer(logger)

	// Simulate the sequencer goroutine holding the action lock mid-action.
	seq.l.Lock()
	defer seq.l.Unlock()

	done := make(chan struct{})
	go func() {
		seq.OnEvent(context.Background(), engine.ForkchoiceUpdateEvent{
			UnsafeL2Head: eth.L2BlockRef{Hash: common.Hash{0xa1}, Number: 1},
		})
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("OnEvent blocked while the sequencer action lock was held")
	}
	require.Len(t, seq.inbox, 1, "event was ingested without the action lock")
}

// runLoopFixture wires a sequencer with a scripted engine for goroutine-level
// RunLoop tests on the real clock.
type runLoopFixture struct {
	seq  *Sequencer
	deps *sequencerTestDeps
	head eth.L2BlockRef
}

func newRunLoopFixture(t *testing.T) *runLoopFixture {
	logger := testlog.Logger(t, log.LevelError)
	seq, deps := createSequencer(logger)
	seq.AttachEmitter(&testutils.MockEmitter{})
	deps.conductor.leader = true

	// Head far enough in the past that build and seal deadlines are due immediately.
	head := eth.L2BlockRef{
		Hash:     common.Hash{0x22},
		Number:   100,
		L1Origin: eth.BlockID{Hash: common.Hash{0x11, 0xa}, Number: 1000},
		Time:     uint64(time.Now().Add(-10 * time.Second).Unix()),
	}
	l1Origin := eth.L1BlockRef{
		Hash:       common.Hash{0x11, 0xb},
		ParentHash: head.L1Origin.Hash,
		Number:     head.L1Origin.Number + 1,
		Time:       head.Time,
	}
	deps.l1OriginSelector.l1OriginFn = func(eth.L2BlockRef) (eth.L1BlockRef, error) {
		return l1Origin, nil
	}
	return &runLoopFixture{seq: seq, deps: deps, head: head}
}

// startLoop runs RunLoop on its own goroutine and returns a stop function that
// cancels the loop and waits for it to exit.
func (f *runLoopFixture) startLoop(t *testing.T) (stop func()) {
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		f.seq.RunLoop(ctx)
		close(done)
	}()
	return func() {
		cancel()
		select {
		case <-done:
		case <-time.After(10 * time.Second):
			t.Fatal("RunLoop did not exit on context cancellation")
		}
	}
}

// TestSequencerRunLoop_SequencesBlocks is the goroutine-level smoke test: with
// only the run-loop's own timer and wake channel driving it, the sequencer
// builds, seals, and processes a block end-to-end via direct engine calls.
func TestSequencerRunLoop_SequencesBlocks(t *testing.T) {
	f := newRunLoopFixture(t)

	payloadInfo := eth.PayloadInfo{ID: eth.PayloadID{0x42}, Timestamp: f.head.Time + f.deps.cfg.BlockTime}
	envelope := &eth.ExecutionPayloadEnvelope{
		ExecutionPayload: &eth.ExecutionPayload{
			ParentHash:   f.head.Hash,
			BlockNumber:  eth.Uint64Quantity(f.head.Number + 1),
			BlockHash:    common.Hash{0x12, 0x34},
			Timestamp:    eth.Uint64Quantity(f.head.Time + f.deps.cfg.BlockTime),
			Transactions: []eth.Data{encodeID(eth.BlockID{Hash: common.Hash{0x11, 0xb}, Number: 1001})},
		},
	}
	payloadRef := eth.L2BlockRef{
		Hash:       envelope.ExecutionPayload.BlockHash,
		Number:     uint64(envelope.ExecutionPayload.BlockNumber),
		ParentHash: envelope.ExecutionPayload.ParentHash,
		Time:       uint64(envelope.ExecutionPayload.Timestamp),
	}

	f.deps.eng.startBuildFn = func(ctx context.Context, attrs *derive.AttributesWithParent) (*engine.BuildStartResult, error) {
		return &engine.BuildStartResult{Info: payloadInfo, BuildStarted: time.Now(), Parent: attrs.Parent}, nil
	}
	seals := 0 // only called from the loop goroutine
	f.deps.eng.sealBuildFn = func(ctx context.Context, info eth.PayloadInfo, buildStarted time.Time) (*engine.SealResult, error) {
		seals++
		if seals > 1 {
			// After the block under test, park the loop via the error backoff.
			return nil, errors.New("mock seal cap")
		}
		return &engine.SealResult{Envelope: envelope, Ref: payloadRef}, nil
	}
	processed := make(chan eth.L2BlockRef, 16)
	f.deps.eng.processPayloadFn = func(ctx context.Context, envelope *eth.ExecutionPayloadEnvelope, ref eth.L2BlockRef, buildStarted time.Time) error {
		select {
		case processed <- ref:
		default:
		}
		return nil
	}

	require.NoError(t, f.seq.Init(context.Background(), true))
	// Queue the head before starting the loop; the loop-top drain picks it up.
	f.seq.OnEvent(context.Background(), engine.ForkchoiceUpdateEvent{UnsafeL2Head: f.head})

	stop := f.startLoop(t)
	defer stop()

	select {
	case ref := <-processed:
		require.Equal(t, payloadRef, ref, "sequenced the expected block")
	case <-time.After(30 * time.Second):
		t.Fatal("RunLoop did not sequence a block")
	}
}

// TestSequencerRunLoop_StartWakesStopDisarms checks two lifecycle contracts at
// the goroutine level: an RPC start wakes the parked loop so the first action
// runs promptly without any unrelated wakeup, and after an RPC stop the
// previously armed action deadline fires no further actions.
func TestSequencerRunLoop_StartWakesStopDisarms(t *testing.T) {
	f := newRunLoopFixture(t)
	f.head.Time = uint64(time.Now().Unix()) // fresh head; deadline timing is irrelevant here

	started := make(chan struct{}, 16)
	f.deps.eng.startBuildFn = func(ctx context.Context, attrs *derive.AttributesWithParent) (*engine.BuildStartResult, error) {
		select {
		case started <- struct{}{}:
		default:
		}
		// Fail the build: this re-arms the schedule one block-time (2s) out,
		// leaving a stale deadline behind for the post-Stop phase.
		return nil, errors.New("mock start failure")
	}

	require.NoError(t, f.seq.Init(context.Background(), false))
	deliver(f.seq, engine.ForkchoiceUpdateEvent{UnsafeL2Head: f.head})
	f.seq.latestSealed = f.head

	stop := f.startLoop(t)
	defer stop()

	// The loop is parked: inactive, timer disarmed. Starting the sequencer must
	// wake it and run the first action promptly — no other event source exists
	// in this test that could rouse the loop.
	require.NoError(t, f.seq.Start(context.Background(), f.head.Hash))
	select {
	case <-started:
	case <-time.After(15 * time.Second):
		t.Fatal("sequencer start did not wake the run-loop into action")
	}

	// Stop the sequencer. The schedule was re-armed ~2s out by the failed build;
	// that stale deadline must not produce an action anymore.
	_, err := f.seq.Stop(context.Background())
	require.NoError(t, err)
	// Actions that ran before Stop completed have already signaled: RunAction
	// holds the action lock, and Stop cannot complete while it is held.
	for {
		select {
		case <-started:
			continue
		default:
		}
		break
	}
	select {
	case <-started:
		t.Fatal("stale scheduled action fired after Stop")
	case <-time.After(3 * time.Second): // comfortably past the stale 2s deadline
	}
}

// TestSequencerRunLoop_NoLostWake hammers the wake channel with head updates
// racing the loop's drain/re-plan cycle: the final head must always be
// observed, i.e. no wake signal is lost between drainInbox and the select.
func TestSequencerRunLoop_NoLostWake(t *testing.T) {
	f := newRunLoopFixture(t)

	stop := f.startLoop(t)
	defer stop()

	const numHeads = 200
	for i := 1; i <= numHeads; i++ {
		f.seq.OnEvent(context.Background(), engine.ForkchoiceUpdateEvent{
			UnsafeL2Head: eth.L2BlockRef{
				Hash:   common.Hash{byte(i), byte(i >> 8), 0xff},
				Number: uint64(i),
			},
		})
	}

	// The inactive sequencer records each replayed head. Adjacent updates may
	// coalesce to the latest, but the final head must eventually be applied.
	deadline := time.Now().Add(30 * time.Second)
	for {
		f.seq.l.Lock()
		got := f.seq.latestHead.Number
		f.seq.l.Unlock()
		if got == numHeads {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("lost wake: run-loop stalled at head %d of %d", got, numHeads)
		}
		time.Sleep(2 * time.Millisecond)
	}
}

// TestSequencerRunLoop_DeadlineMovedByDrain covers the timer/ingest race: an
// event can land after the loop armed its timer, and the drain that follows the
// timer fire may push the deadline out (engine backoff, post-reset delay). The
// loop must re-plan on the new deadline instead of acting on the fired one,
// which would defeat the backoff it just applied.
func TestSequencerRunLoop_DeadlineMovedByDrain(t *testing.T) {
	f := newRunLoopFixture(t)

	var builds atomic.Int64
	f.deps.eng.startBuildFn = func(context.Context, *derive.AttributesWithParent) (*engine.BuildStartResult, error) {
		builds.Add(1)
		return nil, errors.New("must not be called")
	}

	f.seq.l.Lock()
	f.seq.active.Store(true)
	f.seq.nextActionOK = true
	f.seq.nextAction = time.Now().Add(500 * time.Millisecond)
	f.seq.l.Unlock()

	stop := f.startLoop(t)
	defer stop()
	f.seq.wake() // re-plan onto the deadline set above

	// Append without waking: the event must be picked up by the drain that
	// follows the timer fire, which is the interleaving under test.
	time.Sleep(100 * time.Millisecond)
	f.seq.inboxMu.Lock()
	f.seq.inbox = append(f.seq.inbox, rollup.EngineTemporaryErrorEvent{Err: engine.ErrEngineSyncing})
	f.seq.inboxMu.Unlock()

	// Wait well past the original deadline.
	time.Sleep(700 * time.Millisecond)
	require.Zero(t, builds.Load(), "the fired deadline was superseded by the backoff")
	next, ok := f.seq.NextAction()
	require.True(t, ok)
	require.Greater(t, time.Until(next), 20*time.Second, "the syncing backoff is in force")
}

// TestSequencerRunLoop_NoSpinOnUnchangedSchedule pins the rule that the loop
// never re-arms a deadline it already fired. An action that returns without
// changing the schedule or clearing nextActionOK would otherwise re-arm the
// same past deadline and spin at full CPU, holding the sequencer lock and
// starving Start/Stop. No such action exists today; this guards future ones,
// so the no-op action is constructed directly.
func TestSequencerRunLoop_NoSpinOnUnchangedSchedule(t *testing.T) {
	f := newRunLoopFixture(t)

	var clockReads atomic.Int64
	f.seq.timeNow = func() time.Time {
		clockReads.Add(1)
		return time.Now()
	}

	// Inactive with an armed, already-due schedule: RunAction returns at the
	// inactive check without touching the schedule.
	f.seq.l.Lock()
	f.seq.nextActionOK = true
	f.seq.nextAction = time.Now().Add(-time.Second)
	f.seq.l.Unlock()

	stop := f.startLoop(t)
	defer stop()

	time.Sleep(100 * time.Millisecond)
	require.Less(t, clockReads.Load(), int64(10),
		"loop re-fired the same deadline instead of parking")
}
