package sequencing

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"math/rand" // nosemgrep
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-node/metrics"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/conductor"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-node/rollup/engine"
	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/event"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
)

type FakeAttributesBuilder struct {
	cfg *rollup.Config
	rng *rand.Rand
}

// used to put the L1 origin into the data-tx, without all the deposit-tx complexity, for testing purposes.
func encodeID(id eth.BlockID) []byte {
	var out [32 + 8]byte
	copy(out[:32], id.Hash[:])
	binary.BigEndian.PutUint64(out[32:], id.Number)
	return out[:]
}

func decodeID(data []byte) eth.BlockID {
	return eth.BlockID{
		Hash:   common.Hash(data[:32]),
		Number: binary.BigEndian.Uint64(data[32:]),
	}
}

func (m *FakeAttributesBuilder) PreparePayloadAttributes(ctx context.Context,
	l2Parent eth.L2BlockRef, epoch eth.BlockID,
) (attrs *eth.PayloadAttributes, err error) {
	gasLimit := eth.Uint64Quantity(30_000_000)
	attrs = &eth.PayloadAttributes{
		Timestamp:             eth.Uint64Quantity(l2Parent.Time + m.cfg.BlockTime),
		PrevRandao:            eth.Bytes32(testutils.RandomHash(m.rng)),
		SuggestedFeeRecipient: predeploys.SequencerFeeVaultAddr,
		Withdrawals:           nil,
		ParentBeaconBlockRoot: nil,
		Transactions:          []eth.Data{encodeID(epoch)}, // simplified replacement for L1-info tx.
		NoTxPool:              false,
		GasLimit:              &gasLimit,
	}

	if m.cfg.IsEcotone(uint64(attrs.Timestamp)) {
		r := testutils.RandomHash(m.rng)
		attrs.ParentBeaconBlockRoot = &r
	}
	return attrs, nil
}

var _ derive.AttributesBuilder = (*FakeAttributesBuilder)(nil)

type FakeL1OriginSelector struct {
	request    eth.L2BlockRef
	l1OriginFn func(l2Head eth.L2BlockRef) (eth.L1BlockRef, error)
}

func (f *FakeL1OriginSelector) FindL1Origin(ctx context.Context, l2Head eth.L2BlockRef) (eth.L1BlockRef, error) {
	f.request = l2Head
	return f.l1OriginFn(l2Head)
}

func (f *FakeL1OriginSelector) SetRecoverMode(bool) {
	// noop
}

var _ L1OriginSelectorIface = (*FakeL1OriginSelector)(nil)

type BasicSequencerStateListener struct {
	active bool
}

func (b *BasicSequencerStateListener) SequencerStarted() error {
	b.active = true
	return nil
}

func (b *BasicSequencerStateListener) SequencerStopped() error {
	b.active = false
	return nil
}

var _ SequencerStateListener = (*BasicSequencerStateListener)(nil)

// FakeConductor is a no-op conductor that assumes this node is the leader sequencer.
type FakeConductor struct {
	closed    bool
	leader    bool
	committed *eth.ExecutionPayloadEnvelope
}

var _ conductor.SequencerConductor = &FakeConductor{}

func (c *FakeConductor) Enabled(ctx context.Context) bool {
	return true
}

func (c *FakeConductor) Leader(ctx context.Context) (bool, error) {
	return c.leader, nil
}

func (c *FakeConductor) CommitUnsafePayload(ctx context.Context, payload *eth.ExecutionPayloadEnvelope) error {
	c.committed = payload
	return nil
}

func (c *FakeConductor) OverrideLeader(ctx context.Context) error {
	c.leader = true
	return nil
}

func (c *FakeConductor) Close() {
	c.closed = true
}

type FakeAsyncGossip struct {
	payload *eth.ExecutionPayloadEnvelope
	started bool
	stopped bool
}

func (f *FakeAsyncGossip) Gossip(payload *eth.ExecutionPayloadEnvelope) {
	f.payload = payload
}

func (f *FakeAsyncGossip) Get() *eth.ExecutionPayloadEnvelope {
	return f.payload
}

func (f *FakeAsyncGossip) Clear() {
	f.payload = nil
}

func (f *FakeAsyncGossip) Stop() {
	f.stopped = true
}

func (f *FakeAsyncGossip) Start() {
	f.started = true
}

var _ AsyncGossiper = (*FakeAsyncGossip)(nil)

// fakeEngController is a scripted SequencerEngine: tests configure the
// direct-call results per method. Unconfigured methods fail with an error,
// so unexpected engine calls surface as sequencing errors.
type fakeEngController struct {
	fcuRequests int

	startBuildFn     func(ctx context.Context, attrs *derive.AttributesWithParent) (*engine.BuildStartResult, error)
	sealBuildFn      func(ctx context.Context, info eth.PayloadInfo, buildStarted time.Time) (*engine.SealResult, error)
	processPayloadFn func(ctx context.Context, envelope *eth.ExecutionPayloadEnvelope, ref eth.L2BlockRef, buildStarted time.Time) error
}

func (f *fakeEngController) RequestForkchoiceUpdate(ctx context.Context) {
	f.fcuRequests++
}

func (f *fakeEngController) StartBuild(ctx context.Context, attrs *derive.AttributesWithParent) (*engine.BuildStartResult, error) {
	if f.startBuildFn != nil {
		return f.startBuildFn(ctx, attrs)
	}
	return nil, errors.New("StartBuild not configured")
}

func (f *fakeEngController) SealBuild(ctx context.Context, info eth.PayloadInfo, buildStarted time.Time) (*engine.SealResult, error) {
	if f.sealBuildFn != nil {
		return f.sealBuildFn(ctx, info, buildStarted)
	}
	return nil, errors.New("SealBuild not configured")
}

func (f *fakeEngController) ProcessPayload(ctx context.Context, envelope *eth.ExecutionPayloadEnvelope, ref eth.L2BlockRef, buildStarted time.Time) error {
	if f.processPayloadFn != nil {
		return f.processPayloadFn(ctx, envelope, ref, buildStarted)
	}
	return errors.New("ProcessPayload not configured")
}

var _ SequencerEngine = (*fakeEngController)(nil)

// deliver feeds an event through OnEvent and synchronously replays the inbox,
// standing in for the sequencer goroutine's loop-top drain.
func deliver(seq *Sequencer, ev event.Event) {
	seq.OnEvent(context.Background(), ev)
	seq.drainInbox()
}

// TestSequencer_StartStop runs through start/stop state back and forth to test state changes.
func TestSequencer_StartStop(t *testing.T) {
	logger := testlog.Logger(t, log.LevelError)
	seq, deps := createSequencer(logger)
	emitter := &testutils.MockEmitter{}
	seq.AttachEmitter(emitter)

	// Allow the sequencer to be the leader.
	// This is checked, since we start sequencing later, after initialization.
	// Also see issue #11121 for context: the conductor is checked by the infra, when initialized in active state.
	deps.conductor.leader = true

	testCtx := context.Background()
	require.NoError(t, seq.Init(testCtx, false))
	emitter.AssertExpectations(t)
	require.False(t, deps.conductor.closed, "conductor is ready")
	require.True(t, deps.asyncGossip.started, "async gossip is always started on initialization")
	require.False(t, deps.seqState.active, "sequencer not active yet")

	require.Equal(t, common.Hash{}, seq.building.Ref.Hash)
	require.Equal(t, common.Hash{}, seq.lastSealed.Hash)
	require.Equal(t, common.Hash{}, seq.unsafeHead.Hash)

	// Set unsafeHead via a forkchoice update through the inbox
	deliver(seq, engine.ForkchoiceUpdateEvent{
		UnsafeL2Head:    eth.L2BlockRef{Hash: common.Hash{0xaa}},
		SafeL2Head:      eth.L2BlockRef{},
		FinalizedL2Head: eth.L2BlockRef{},
	})
	require.Equal(t, common.Hash{0xaa}, seq.unsafeHead.Hash)
	// Sealing happens via direct engine calls now; no sealed block exists in
	// this test, so match lastSealed to unsafeHead manually to keep Stop()
	// from waiting for the (nonexistent) sealed block to become canonical.
	seq.lastSealed = eth.L2BlockRef{Hash: common.Hash{0xaa}}

	require.False(t, seq.Active())
	// no action scheduled
	_, ok := seq.NextAction()
	require.False(t, ok)

	require.NoError(t, seq.Start(context.Background(), common.Hash{0xaa}))
	require.True(t, seq.Active())
	require.True(t, deps.seqState.active, "sequencer signaled it is active")

	// sequencer is active now, it should schedule work
	_, ok = seq.NextAction()
	require.True(t, ok)

	// can't activate again before stopping
	err := seq.Start(context.Background(), common.Hash{0xaa})
	require.ErrorIs(t, err, ErrSequencerAlreadyStarted)

	head, err := seq.Stop(context.Background())
	require.NoError(t, err)
	require.Equal(t, head, common.Hash{0xaa})
	require.False(t, deps.seqState.active, "sequencer signaled it is no longer active")

	_, err = seq.Stop(context.Background())
	require.ErrorIs(t, err, ErrSequencerAlreadyStopped)

	// need to resume from the last head
	err = seq.Start(context.Background(), common.Hash{0xbb})
	require.ErrorContains(t, err, "block hash does not match")

	// can start again from head that it left
	err = seq.Start(context.Background(), head)
	require.NoError(t, err)
}

// TestSequencer_NoActionAfterStop verifies that a sequencer action that fires
// after Stop() (because the loop's timer was already armed before we deactivated)
// is ignored, rather than starting a new block-building job. Acting on that stale
// deadline would issue a forkchoice update reasserting our head and could undo a
// reorg introduced externally after we stopped. Regression test for issue #20198.
func TestSequencer_NoActionAfterStop(t *testing.T) {
	logger := testlog.Logger(t, log.LevelError)
	seq, deps := createSequencer(logger)
	emitter := &testutils.MockEmitter{}
	seq.AttachEmitter(emitter)
	deps.conductor.leader = true

	testCtx := context.Background()
	require.NoError(t, seq.Init(testCtx, false))
	emitter.AssertExpectations(t)
	fcuRequestsAfterInit := deps.eng.fcuRequests

	// Bring lastSealed and unsafeHead to the same known block, so Stop() does not
	// block waiting for the sealed block to catch up to the head.
	head := eth.L2BlockRef{Hash: common.Hash{0xaa}}
	deliver(seq, engine.ForkchoiceUpdateEvent{UnsafeL2Head: head})
	require.Equal(t, head.Hash, seq.unsafeHead.Hash)
	seq.lastSealed = head

	// Start, then stop. While active the sequencer scheduled work; the loop may
	// have already armed the action timer at this point.
	require.NoError(t, seq.Start(testCtx, head.Hash))
	require.True(t, seq.Active())
	_, ok := seq.NextAction()
	require.True(t, ok, "active sequencer schedules work")

	stopHead, err := seq.Stop(testCtx)
	require.NoError(t, err)
	require.Equal(t, head.Hash, stopHead)
	require.False(t, seq.Active())

	// Provide a valid L1 origin so that, absent the inactive-guard, RunAction
	// would proceed all the way to a StartBuild call — turning a regression
	// into a clear test failure.
	deps.l1OriginSelector.l1OriginFn = func(l2Head eth.L2BlockRef) (eth.L1BlockRef, error) {
		return eth.L1BlockRef{Number: 1000}, nil
	}
	deps.eng.startBuildFn = func(ctx context.Context, attrs *derive.AttributesWithParent) (*engine.BuildStartResult, error) {
		t.Error("StartBuild must not be called after Stop")
		return nil, errors.New("unexpected StartBuild")
	}

	// Fire the stale action. The sequencer must not act while stopped.
	seq.RunAction()
	emitter.AssertExpectations(t)

	require.Equal(t, BuildingState{}, seq.building, "no block-building job started while stopped")
	require.Equal(t, fcuRequestsAfterInit, deps.eng.fcuRequests, "no forkchoice update requested while stopped")
	_, ok = seq.NextAction()
	require.False(t, ok, "stopped sequencer schedules no further work")
}

// TestSequencer_StaleBuild stops the sequencer after block-building,
// but before processing the block locally,
// and then continues it again, to check if the async-gossip gets cleared,
// instead of trying to re-insert the block.
func TestSequencer_StaleBuild(t *testing.T) {
	logger := testlog.Logger(t, log.LevelError)
	seq, deps := createSequencer(logger)

	testClock := clock.NewSimpleClock()
	seq.timeNow = testClock.Now
	testClock.SetTime(30000)

	emitter := &testutils.MockEmitter{}
	seq.AttachEmitter(emitter)
	deps.conductor.leader = true

	testCtx := context.Background()
	require.NoError(t, seq.Init(testCtx, false))
	emitter.AssertExpectations(t)
	require.False(t, deps.conductor.closed, "conductor is ready")
	require.True(t, deps.asyncGossip.started, "async gossip is always started on initialization")
	require.False(t, deps.seqState.active, "sequencer not active yet")

	head := eth.L2BlockRef{
		Hash:   common.Hash{0x22},
		Number: 100,
		L1Origin: eth.BlockID{
			Hash:   common.Hash{0x11, 0xa},
			Number: 1000,
		},
		Time: uint64(testClock.Now().Unix()),
	}
	deliver(seq, engine.ForkchoiceUpdateEvent{UnsafeL2Head: head})

	require.NoError(t, seq.Start(context.Background(), head.Hash))
	require.True(t, seq.Active())
	require.True(t, deps.seqState.active, "sequencer signaled it is active")

	// sequencer is active now, wants to build.
	_, ok := seq.NextAction()
	require.True(t, ok)

	// pretend we progress to the next L1 origin, catching up with the L2 time
	l1Origin := eth.L1BlockRef{
		Hash:       common.Hash{0x11, 0xb},
		ParentHash: head.L1Origin.Hash,
		Number:     head.L1Origin.Number + 1,
		Time:       head.Time + 2,
	}
	deps.l1OriginSelector.l1OriginFn = func(l2Head eth.L2BlockRef) (eth.L1BlockRef, error) {
		return l1Origin, nil
	}

	// pretend we are already 150ms into the block-window when starting building
	startedTime := time.Unix(int64(head.Time), 0).Add(time.Millisecond * 150)
	payloadInfo := eth.PayloadInfo{
		ID:        eth.PayloadID{0x42},
		Timestamp: head.Time + deps.cfg.BlockTime,
	}
	deps.eng.startBuildFn = func(ctx context.Context, attrs *derive.AttributesWithParent) (*engine.BuildStartResult, error) {
		require.Equal(t, head, attrs.Parent)
		require.Equal(t, head.Time+deps.cfg.BlockTime, uint64(attrs.Attributes.Timestamp))
		require.Equal(t, eth.L1BlockRef{}, attrs.DerivedFrom)
		testClock.Set(startedTime)
		return &engine.BuildStartResult{
			Info:         payloadInfo,
			BuildStarted: startedTime,
			Parent:       head,
		}, nil
	}

	// First action: start building. Direct engine call, no events.
	seq.RunAction()
	require.Equal(t, payloadInfo, seq.building.Info, "must have recorded payload info of the started job")

	_, ok = seq.NextAction()
	require.True(t, ok, "must be ready to seal the block now")

	payloadEnvelope := &eth.ExecutionPayloadEnvelope{
		ExecutionPayload: &eth.ExecutionPayload{
			ParentHash:   head.Hash,
			BlockNumber:  eth.Uint64Quantity(head.Number + 1),
			BlockHash:    common.Hash{0x12, 0x34},
			Timestamp:    eth.Uint64Quantity(head.Time + deps.cfg.BlockTime),
			Transactions: []eth.Data{encodeID(l1Origin.ID())},
		},
	}
	payloadRef := eth.L2BlockRef{
		Hash:           payloadEnvelope.ExecutionPayload.BlockHash,
		Number:         uint64(payloadEnvelope.ExecutionPayload.BlockNumber),
		ParentHash:     payloadEnvelope.ExecutionPayload.ParentHash,
		Time:           uint64(payloadEnvelope.ExecutionPayload.Timestamp),
		L1Origin:       l1Origin.ID(),
		SequenceNumber: 0,
	}
	deps.eng.sealBuildFn = func(ctx context.Context, info eth.PayloadInfo, buildStarted time.Time) (*engine.SealResult, error) {
		require.Equal(t, payloadInfo, info)
		return &engine.SealResult{
			Envelope: payloadEnvelope,
			Ref:      payloadRef,
		}, nil
	}
	// The block seals fine, but local processing fails with a temporary
	// (non-sentinel) error: the block is committed and gossiped, but not canonical.
	processPayloadCalled := false
	deps.eng.processPayloadFn = func(ctx context.Context, envelope *eth.ExecutionPayloadEnvelope, ref eth.L2BlockRef, buildStarted time.Time) error {
		processPayloadCalled = true
		return fmt.Errorf("mock temporary engine error")
	}

	// Seal action: SealBuild succeeds, commits to conductor and gossip, ProcessPayload fails.
	seq.RunAction()
	require.True(t, processPayloadCalled, "ProcessPayload must have been called")
	require.Equal(t, payloadEnvelope, deps.conductor.committed, "must commit to conductor")
	require.Equal(t, payloadEnvelope, deps.asyncGossip.payload, "must send to async gossip")
	_, ok = seq.NextAction()
	require.False(t, ok, "not ready to act; the engine's temporary-error event re-arms the schedule via the inbox")

	// attempting to stop block building here should timeout, because the sealed block is different from the unsafeHead
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	_, err := seq.Stop(ctx)
	require.Error(t, err, "stop should have timed out")
	require.ErrorIs(t, err, ctx.Err())

	// Now let's say another sequencer built a bunch of blocks,
	// can we continue from there? We'll have to wipe the old in-flight block,
	// if we continue on top of a chain that had it already included a while ago.

	// Signal the new chain we are building on
	testClock.Set(testClock.Now().Add(time.Second * 100 * 2))

	newL1Origin := eth.L1BlockRef{
		Hash:       common.Hash{0x11, 0x11, 0x44},
		ParentHash: head.L1Origin.Hash,
		Number:     head.L1Origin.Number + 50,
		Time:       uint64(testClock.Now().Unix()),
	}
	newHead := eth.L2BlockRef{
		Hash:     common.Hash{0x44},
		Number:   head.Number + 100,
		L1Origin: newL1Origin.ID(),
		Time:     uint64(testClock.Now().Unix()),
	}
	deliver(seq, engine.ForkchoiceUpdateEvent{UnsafeL2Head: newHead})
	_, ok = seq.NextAction()
	require.True(t, ok, "new head re-arms the sequencer")

	// Regression check: async-gossip is cleared upon sequencer un-pause.
	// We could clear it earlier. But absolutely have to clear it upon Start(),
	// to not continue from this older point.
	require.NotNil(t, deps.asyncGossip.payload, "async-gossip still not cleared")

	// Stop() waits for lastSealed == unsafeHead. The head advanced past our
	// sealed block (another sequencer took over), so match lastSealed to
	// unblock Stop().
	seq.lastSealed = newHead
	stopHead, err := seq.Stop(context.Background())
	require.NoError(t, err)
	require.Equal(t, newHead.Hash, stopHead)
	require.False(t, deps.seqState.active, "sequencer signaled it is no longer active")

	// start sequencing on top of the new chain
	require.NoError(t, seq.Start(context.Background(), newHead.Hash), "must continue from new block")

	// regression check: no stale async gossip is continued
	require.Nil(t, deps.asyncGossip.payload, "async gossip should be cleared on Start")

	// Start building the block with the new L1 origin
	deps.l1OriginSelector.l1OriginFn = func(l2Head eth.L2BlockRef) (eth.L1BlockRef, error) {
		return newL1Origin, nil
	}
	_, ok = seq.NextAction()
	require.True(t, ok, "ready to sequence again")

	// Sequencer action must start a fresh build on top of the new head,
	// and not try to seal what was previously in-flight.
	deps.eng.startBuildFn = func(ctx context.Context, attrs *derive.AttributesWithParent) (*engine.BuildStartResult, error) {
		require.Equal(t, newHead, attrs.Parent, "build on the new L2 head")
		return &engine.BuildStartResult{
			Info:         eth.PayloadInfo{ID: eth.PayloadID{0x99}, Timestamp: newHead.Time + deps.cfg.BlockTime},
			BuildStarted: testClock.Now(),
			Parent:       newHead,
		}, nil
	}
	deps.eng.sealBuildFn = nil
	seq.RunAction()
	require.Equal(t, newHead, seq.building.Onto, "must build on the new head")
}

func TestSequencerBuild(t *testing.T) {
	logger := testlog.Logger(t, log.LevelError)
	seq, deps := createSequencer(logger)
	testClock := clock.NewSimpleClock()
	seq.timeNow = testClock.Now
	testClock.SetTime(30000)
	emitter := &testutils.MockEmitter{}
	seq.AttachEmitter(emitter)

	testCtx := context.Background()
	// Init requests a forkchoice update directly from the engine
	require.NoError(t, seq.Init(testCtx, true))
	require.True(t, seq.Active(), "started in active mode")
	require.Equal(t, 1, deps.eng.fcuRequests)

	// Without a known head there is nothing to build on; the sequencer requests
	// a forkchoice update to learn the head, and parks until it arrives —
	// re-arming the unchanged deadline would hot-loop forkchoice requests.
	seq.RunAction()
	require.Equal(t, 2, deps.eng.fcuRequests)
	_, ok := seq.NextAction()
	require.False(t, ok, "parked until the requested head update arrives")

	// Now send the forkchoice data, for the sequencer to learn what to build on top of.
	head := eth.L2BlockRef{
		Hash:   common.Hash{0x22},
		Number: 100,
		L1Origin: eth.BlockID{
			Hash:   common.Hash{0x11, 0xa},
			Number: 1000,
		},
		Time: uint64(testClock.Now().Unix()),
	}
	deliver(seq, engine.ForkchoiceUpdateEvent{UnsafeL2Head: head})

	// pretend we progress to the next L1 origin, catching up with the L2 time
	l1Origin := eth.L1BlockRef{
		Hash:       common.Hash{0x11, 0xb},
		ParentHash: common.Hash{0x11, 0xa},
		Number:     1001,
		Time:       29998,
	}
	deps.l1OriginSelector.l1OriginFn = func(l2Head eth.L2BlockRef) (eth.L1BlockRef, error) {
		return l1Origin, nil
	}

	// pretend we are already 150ms into the block-window when starting building
	startedTime := time.Unix(int64(head.Time), 0).Add(time.Millisecond * 150)
	payloadInfo := eth.PayloadInfo{
		ID:        eth.PayloadID{0x42},
		Timestamp: head.Time + deps.cfg.BlockTime,
	}
	var sentAttributes *derive.AttributesWithParent
	deps.eng.startBuildFn = func(ctx context.Context, attrs *derive.AttributesWithParent) (*engine.BuildStartResult, error) {
		require.Equal(t, head, attrs.Parent)
		require.Equal(t, head.Time+deps.cfg.BlockTime, uint64(attrs.Attributes.Timestamp))
		require.Equal(t, eth.L1BlockRef{}, attrs.DerivedFrom)
		sentAttributes = attrs
		testClock.Set(startedTime)
		return &engine.BuildStartResult{
			Info:         payloadInfo,
			BuildStarted: startedTime,
			Parent:       head,
		}, nil
	}

	// The action starts building via a direct StartBuild call.
	seq.RunAction()
	require.NotNil(t, sentAttributes, "StartBuild must have been called")
	require.Equal(t, payloadInfo, seq.building.Info)

	// The sealing should now be scheduled as next action.
	// We expect to seal just before the block-time boundary, leaving enough time for the sealing itself.
	sealTargetTime, ok := seq.NextAction()
	require.True(t, ok)
	buildDuration := sealTargetTime.Sub(time.Unix(int64(head.Time), 0))
	require.Equal(t, (time.Duration(deps.cfg.BlockTime)*time.Second)-defaultSealingDuration, buildDuration)

	payloadEnvelope := &eth.ExecutionPayloadEnvelope{
		ParentBeaconBlockRoot: sentAttributes.Attributes.ParentBeaconBlockRoot,
		ExecutionPayload: &eth.ExecutionPayload{
			ParentHash:   head.Hash,
			FeeRecipient: sentAttributes.Attributes.SuggestedFeeRecipient,
			BlockNumber:  eth.Uint64Quantity(sentAttributes.Parent.Number + 1),
			BlockHash:    common.Hash{0x12, 0x34},
			Timestamp:    sentAttributes.Attributes.Timestamp,
			Transactions: sentAttributes.Attributes.Transactions,
			// Not all attributes matter to sequencer. We can leave these nil.
		},
	}
	payloadRef := eth.L2BlockRef{
		Hash:           payloadEnvelope.ExecutionPayload.BlockHash,
		Number:         uint64(payloadEnvelope.ExecutionPayload.BlockNumber),
		ParentHash:     payloadEnvelope.ExecutionPayload.ParentHash,
		Time:           uint64(payloadEnvelope.ExecutionPayload.Timestamp),
		L1Origin:       l1Origin.ID(),
		SequenceNumber: 0,
	}
	deps.eng.sealBuildFn = func(ctx context.Context, info eth.PayloadInfo, buildStarted time.Time) (*engine.SealResult, error) {
		require.Equal(t, payloadInfo, info)
		require.Equal(t, startedTime, buildStarted)
		return &engine.SealResult{
			Envelope: payloadEnvelope,
			Ref:      payloadRef,
		}, nil
	}
	deps.eng.processPayloadFn = func(ctx context.Context, envelope *eth.ExecutionPayloadEnvelope, ref eth.L2BlockRef, buildStarted time.Time) error {
		require.Equal(t, payloadEnvelope, envelope)
		require.Equal(t, payloadRef, ref)
		require.Equal(t, startedTime, buildStarted)
		return nil
	}

	// Seal action: seals, commits to conductor, gossips, and inserts the payload,
	// all synchronously via direct calls.
	seq.RunAction()
	require.Equal(t, payloadEnvelope, deps.conductor.committed, "must commit to conductor")
	require.Nil(t, deps.asyncGossip.payload, "async gossip should have cleared after successful insert")
	require.Equal(t, BuildingState{}, seq.building, "building state cleared after successful insert")
	require.Equal(t, payloadRef, seq.lastSealed, "sealed block recorded")

	// After a successful insert the sequencer schedules the next block itself,
	// without waiting for the forkchoice update to travel through the event system.
	nextTime, ok := seq.NextAction()
	require.True(t, ok, "ready to build next block")
	require.Equal(t, payloadRef, seq.unsafeHead, "unsafeHead updated directly after successful insert")
	require.Equal(t, time.Unix(int64(payloadRef.Time), 0), nextTime,
		"next build starts a block-time before the next payload deadline")

	// The engine's ForkchoiceUpdateEvent still arrives later via the inbox;
	// it is idempotent since the head was already updated.
	testClock.Set(time.Unix(int64(payloadRef.Time), 0).Add(time.Millisecond * 120))
	deliver(seq, engine.ForkchoiceUpdateEvent{
		UnsafeL2Head:    payloadRef,
		SafeL2Head:      eth.L2BlockRef{},
		FinalizedL2Head: eth.L2BlockRef{},
	})
	_, ok = seq.NextAction()
	require.True(t, ok, "still ready after idempotent forkchoice update")
}

func TestSequencerL1TemporaryErrorEvent(t *testing.T) {
	logger := testlog.Logger(t, log.LevelError)
	seq, deps := createSequencer(logger)
	testClock := clock.NewSimpleClock()
	seq.timeNow = testClock.Now
	testClock.SetTime(30000)
	emitter := &testutils.MockEmitter{}
	seq.AttachEmitter(emitter)

	testCtx := context.Background()
	// Init
	require.NoError(t, seq.Init(testCtx, true))
	require.True(t, seq.Active(), "started in active mode")

	// It needs the head before being able to build on top of it
	seq.RunAction()

	// Now send the forkchoice data, for the sequencer to learn what to build on top of.
	head := eth.L2BlockRef{
		Hash:   common.Hash{0x22},
		Number: 100,
		L1Origin: eth.BlockID{
			Hash:   common.Hash{0x11, 0xa},
			Number: 1000,
		},
		Time: uint64(testClock.Now().Unix()),
	}
	deliver(seq, engine.ForkchoiceUpdateEvent{UnsafeL2Head: head})
	emitter.AssertExpectations(t)

	// force FindL1Origin to return an error
	deps.l1OriginSelector.l1OriginFn = func(l2Head eth.L2BlockRef) (eth.L1BlockRef, error) {
		return eth.L1BlockRef{}, fmt.Errorf("l1OriginFn error")
	}

	emitter.ExpectOnceRun(func(ev event.Event) {
		_, ok := ev.(rollup.L1TemporaryErrorEvent)
		require.True(t, ok)
	})

	sealTargetTime1, ok1 := seq.NextAction()
	seq.RunAction()
	emitter.AssertExpectations(t)

	// FindL1Origin error pushes s.nextAction into the future
	sealTargetTime2, ok2 := seq.NextAction()

	require.True(t, ok1 == ok2 && sealTargetTime2.After(sealTargetTime1))
}

// seqTestSetup is an active sequencer at a known head, with a mocked clock at
// the head timestamp and an L1 origin ready for the next block.
type seqTestSetup struct {
	seq      *Sequencer
	deps     *sequencerTestDeps
	clock    *clock.SimpleClock
	head     eth.L2BlockRef
	l1Origin eth.L1BlockRef
}

func newSeqSetup(t *testing.T) *seqTestSetup {
	logger := testlog.Logger(t, log.LevelError)
	seq, deps := createSequencer(logger)
	testClock := clock.NewSimpleClock()
	testClock.SetTime(30000)
	seq.timeNow = testClock.Now
	seq.AttachEmitter(&testutils.MockEmitter{})
	deps.conductor.leader = true
	require.NoError(t, seq.Init(context.Background(), true))

	head := eth.L2BlockRef{
		Hash:   common.Hash{0x22},
		Number: 100,
		L1Origin: eth.BlockID{
			Hash:   common.Hash{0x11, 0xa},
			Number: 1000,
		},
		Time: uint64(testClock.Now().Unix()),
	}
	deliver(seq, engine.ForkchoiceUpdateEvent{UnsafeL2Head: head})

	l1Origin := eth.L1BlockRef{
		Hash:       common.Hash{0x11, 0xb},
		ParentHash: head.L1Origin.Hash,
		Number:     head.L1Origin.Number + 1,
		Time:       head.Time,
	}
	deps.l1OriginSelector.l1OriginFn = func(eth.L2BlockRef) (eth.L1BlockRef, error) {
		return l1Origin, nil
	}
	return &seqTestSetup{seq: seq, deps: deps, clock: testClock, head: head, l1Origin: l1Origin}
}

// startBuild runs one action that successfully starts a block-building job.
func (s *seqTestSetup) startBuild(t *testing.T) eth.PayloadInfo {
	info := eth.PayloadInfo{ID: eth.PayloadID{0x42}, Timestamp: s.head.Time + s.deps.cfg.BlockTime}
	s.deps.eng.startBuildFn = func(ctx context.Context, attrs *derive.AttributesWithParent) (*engine.BuildStartResult, error) {
		return &engine.BuildStartResult{Info: info, BuildStarted: s.clock.Now(), Parent: attrs.Parent}, nil
	}
	s.seq.RunAction()
	require.Equal(t, info, s.seq.building.Info, "build must have started")
	return info
}

// sealedPayload returns a payload envelope and matching block-ref for the block
// on top of the setup head. The L1 origin is encoded in tx[0], so the test
// toBlockRef hook can reconstruct the same ref from the envelope.
func (s *seqTestSetup) sealedPayload() (*eth.ExecutionPayloadEnvelope, eth.L2BlockRef) {
	envelope := &eth.ExecutionPayloadEnvelope{
		ExecutionPayload: &eth.ExecutionPayload{
			ParentHash:   s.head.Hash,
			BlockNumber:  eth.Uint64Quantity(s.head.Number + 1),
			BlockHash:    common.Hash{0x12, 0x34},
			Timestamp:    eth.Uint64Quantity(s.head.Time + s.deps.cfg.BlockTime),
			Transactions: []eth.Data{encodeID(s.l1Origin.ID())},
		},
	}
	ref := eth.L2BlockRef{
		Hash:           envelope.ExecutionPayload.BlockHash,
		Number:         uint64(envelope.ExecutionPayload.BlockNumber),
		ParentHash:     envelope.ExecutionPayload.ParentHash,
		Time:           uint64(envelope.ExecutionPayload.Timestamp),
		L1Origin:       s.l1Origin.ID(),
		SequenceNumber: 0,
	}
	return envelope, ref
}

func (s *seqTestSetup) blockTime() time.Duration {
	return time.Duration(s.deps.cfg.BlockTime) * time.Second
}

// TestSequencerStartBuildErrors covers the direct-call StartBuild error paths.
func TestSequencerStartBuildErrors(t *testing.T) {
	t.Run("stale build", func(t *testing.T) {
		s := newSeqSetup(t)
		// The engine rejects the build because the parent is not the unsafe head
		// anymore. It has already requested a forkchoice update; the sequencer
		// must pause until that head update arrives, instead of hot-retrying.
		s.deps.eng.startBuildFn = func(ctx context.Context, attrs *derive.AttributesWithParent) (*engine.BuildStartResult, error) {
			return nil, engine.ErrStaleBuild
		}
		s.seq.RunAction()
		require.Equal(t, BuildingState{}, s.seq.building, "stale build leaves no building state")
		_, ok := s.seq.NextAction()
		require.False(t, ok, "no action until the next head update")

		// The head update the engine requested arrives and re-arms the sequencer.
		newHead := eth.L2BlockRef{
			Hash:     common.Hash{0x23},
			Number:   s.head.Number + 1,
			L1Origin: s.head.L1Origin,
			Time:     s.head.Time + s.deps.cfg.BlockTime,
		}
		deliver(s.seq, engine.ForkchoiceUpdateEvent{UnsafeL2Head: newHead})
		_, ok = s.seq.NextAction()
		require.True(t, ok, "head update resumes sequencing")
	})

	t.Run("invalid attributes", func(t *testing.T) {
		s := newSeqSetup(t)
		s.deps.eng.startBuildFn = func(ctx context.Context, attrs *derive.AttributesWithParent) (*engine.BuildStartResult, error) {
			return nil, fmt.Errorf("%w: mock payload rejection", engine.ErrBuildInvalid)
		}
		s.seq.RunAction()
		require.Equal(t, BuildingState{}, s.seq.building)
		next, ok := s.seq.NextAction()
		require.True(t, ok, "retry after backoff; no recovery echo exists for invalid attributes")
		require.Equal(t, s.clock.Now().Add(s.blockTime()), next, "back off one block time")
	})

	t.Run("temporary failure", func(t *testing.T) {
		s := newSeqSetup(t)
		mockErr := errors.New("mock temporary start failure")
		s.deps.eng.startBuildFn = func(ctx context.Context, attrs *derive.AttributesWithParent) (*engine.BuildStartResult, error) {
			// The real engine emits EngineTemporaryErrorEvent before returning;
			// that echo arrives through the mailbox and re-arms the sequencer.
			return nil, mockErr
		}
		s.seq.RunAction()
		_, ok := s.seq.NextAction()
		require.False(t, ok, "parked until the temporary-error echo re-arms")

		deliver(s.seq, rollup.EngineTemporaryErrorEvent{Err: mockErr})
		require.Equal(t, BuildingState{}, s.seq.building, "no job id to resume, start over")
		next, ok := s.seq.NextAction()
		require.True(t, ok, "temporary-error echo re-arms")
		require.Equal(t, s.clock.Now().Add(time.Second), next, "short backoff for temporary errors")
	})
}

// TestSequencerSealError checks that a failed SealBuild (expired or invalid seal)
// restarts building: nothing is committed or gossiped, and a new build is
// scheduled after a backoff.
func TestSequencerSealError(t *testing.T) {
	s := newSeqSetup(t)
	info := s.startBuild(t)

	s.deps.eng.sealBuildFn = func(ctx context.Context, gotInfo eth.PayloadInfo, buildStarted time.Time) (*engine.SealResult, error) {
		require.Equal(t, info, gotInfo)
		return nil, errors.New("mock seal error: job expired")
	}
	s.seq.RunAction()

	require.Equal(t, BuildingState{}, s.seq.building, "seal failure restarts the build")
	require.Nil(t, s.deps.conductor.committed, "nothing committed to conductor")
	require.Nil(t, s.deps.asyncGossip.payload, "nothing gossiped")
	next, ok := s.seq.NextAction()
	require.True(t, ok)
	require.Equal(t, s.clock.Now().Add(s.blockTime()), next, "back off one block time")
}

// TestSequencerSealStale checks that a stale seal (competing block landed while
// building) drops the job without committing or gossiping, parks the sequencer,
// and resumes on the engine-requested head update.
func TestSequencerSealStale(t *testing.T) {
	s := newSeqSetup(t)
	s.startBuild(t)
	// A competing block landed while building: the engine refuses to seal
	// the stale job. Nothing may be committed or gossiped; the sequencer
	// parks until the engine-requested head update arrives.
	s.deps.eng.sealBuildFn = func(ctx context.Context, info eth.PayloadInfo, buildStarted time.Time) (*engine.SealResult, error) {
		return nil, engine.ErrStaleBuild
	}
	s.seq.RunAction()
	require.Equal(t, BuildingState{}, s.seq.building, "stale job dropped")
	require.Equal(t, eth.L2BlockRef{}, s.seq.lastSealed, "nothing sealed")
	require.Nil(t, s.deps.conductor.committed, "nothing committed to conductor")
	require.Nil(t, s.deps.asyncGossip.payload, "nothing gossiped")
	_, ok := s.seq.NextAction()
	require.False(t, ok, "parked until the next head update")

	deliver(s.seq, engine.ForkchoiceUpdateEvent{UnsafeL2Head: eth.L2BlockRef{
		Hash:     common.Hash{0x77},
		Number:   s.head.Number + 1,
		L1Origin: s.head.L1Origin,
		Time:     s.head.Time + s.deps.cfg.BlockTime,
	}})
	_, ok = s.seq.NextAction()
	require.True(t, ok, "head update resumes sequencing")
}

// TestSequencerProcessPayloadErrors covers the ProcessPayload error paths of the
// seal action: sentinel errors (invalid/denied) drop the payload, while a
// temporary error keeps it in async-gossip so a later action can retry it.
func TestSequencerProcessPayloadErrors(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
		// dropped marks the payload as permanently rejected: gossip and building
		// state are cleared, and a fresh build is scheduled after a backoff.
		dropped bool
	}{
		{name: "payload invalid", err: engine.ErrPayloadInvalid, dropped: true},
		{name: "payload denied", err: engine.ErrPayloadDenied, dropped: true},
		{name: "temporary error", err: errors.New("mock temp engine error"), dropped: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := newSeqSetup(t)
			s.startBuild(t)
			envelope, ref := s.sealedPayload()
			s.deps.eng.sealBuildFn = func(ctx context.Context, info eth.PayloadInfo, buildStarted time.Time) (*engine.SealResult, error) {
				return &engine.SealResult{Envelope: envelope, Ref: ref}, nil
			}
			s.deps.eng.processPayloadFn = func(ctx context.Context, envelope *eth.ExecutionPayloadEnvelope, ref eth.L2BlockRef, buildStarted time.Time) error {
				return tc.err
			}
			s.seq.RunAction()
			require.Equal(t, envelope, s.deps.conductor.committed, "sealed block is committed before processing")

			if tc.dropped {
				require.Nil(t, s.deps.asyncGossip.payload, "rejected payload is dropped from gossip")
				require.Equal(t, BuildingState{}, s.seq.building)
				next, ok := s.seq.NextAction()
				require.True(t, ok, "restart building after backoff")
				require.Equal(t, s.clock.Now().Add(s.blockTime()), next)
				return
			}

			require.Equal(t, envelope, s.deps.asyncGossip.payload, "payload stays in gossip for retry")
			require.Equal(t, ref, s.seq.building.Ref, "building state is kept")
			_, ok := s.seq.NextAction()
			require.False(t, ok, "paused until the engine's temporary-error event re-arms the schedule")

			// The engine emitted an EngineTemporaryErrorEvent; once it arrives
			// through the inbox, the schedule is re-armed with a backoff.
			deliver(s.seq, rollup.EngineTemporaryErrorEvent{Err: tc.err})
			next, ok := s.seq.NextAction()
			require.True(t, ok)
			require.Equal(t, s.clock.Now().Add(time.Second), next, "temporary-error backoff")

			// The retry action processes the gossiped payload instead of re-sealing.
			var retried *eth.ExecutionPayloadEnvelope
			s.deps.eng.processPayloadFn = func(ctx context.Context, envelope *eth.ExecutionPayloadEnvelope, gotRef eth.L2BlockRef, buildStarted time.Time) error {
				retried = envelope
				require.Equal(t, ref, gotRef, "ref reconstructed from the gossiped payload")
				require.True(t, buildStarted.IsZero(), "no build-start time for gossip-buffer retries")
				return nil
			}
			s.seq.RunAction()
			require.Equal(t, envelope, retried, "gossiped payload was retried")
			require.Nil(t, s.deps.asyncGossip.payload, "gossip cleared after successful retry")
			require.Equal(t, BuildingState{}, s.seq.building)
			require.Equal(t, ref, s.seq.unsafeHead, "head updated directly after successful insert")
			_, ok = s.seq.NextAction()
			require.True(t, ok, "ready to build the next block")
		})
	}
}

// TestSequencerMaxSafeLagStall checks that the maxSafeLag stall overrides the
// head-advancement arming within a single forkchoice update, and that sequencing
// resumes once the safe head catches up.
func TestSequencerMaxSafeLagStall(t *testing.T) {
	s := newSeqSetup(t)
	require.NoError(t, s.seq.SetMaxSafeLag(context.Background(), 2))

	// Unsafe head advances, but safe head lags too far: the stall check runs
	// after the head-advance arming and must win.
	deliver(s.seq, engine.ForkchoiceUpdateEvent{
		UnsafeL2Head: eth.L2BlockRef{Hash: common.Hash{0x31}, Number: s.head.Number + 1, Time: s.head.Time + 2},
		SafeL2Head:   eth.L2BlockRef{Hash: common.Hash{0x30}, Number: s.head.Number - 1},
	})
	_, ok := s.seq.NextAction()
	require.False(t, ok, "stalled by max safe lag despite advancing head")

	// Still lagging: stays stalled.
	deliver(s.seq, engine.ForkchoiceUpdateEvent{
		UnsafeL2Head: eth.L2BlockRef{Hash: common.Hash{0x32}, Number: s.head.Number + 2, Time: s.head.Time + 4},
		SafeL2Head:   eth.L2BlockRef{Hash: common.Hash{0x30}, Number: s.head.Number - 1},
	})
	_, ok = s.seq.NextAction()
	require.False(t, ok, "still stalled while safe head lags")

	// Safe head catches up: resume immediately.
	deliver(s.seq, engine.ForkchoiceUpdateEvent{
		UnsafeL2Head: eth.L2BlockRef{Hash: common.Hash{0x33}, Number: s.head.Number + 3, Time: s.head.Time + 6},
		SafeL2Head:   eth.L2BlockRef{Hash: common.Hash{0x32}, Number: s.head.Number + 2},
	})
	next, ok := s.seq.NextAction()
	require.True(t, ok, "resumed after safe head caught up")
	require.Equal(t, s.clock.Now(), next, "resume immediately")
}

// TestSequencerMaxSafeLagDirectPath checks that the maxSafeLag bound is also
// enforced when the schedule advances via the direct-call insertion path, not
// only on ingested forkchoice updates: the sequencer must not outrun the bound
// while the stalling forkchoice echo is still queued behind derivation events.
func TestSequencerMaxSafeLagDirectPath(t *testing.T) {
	s := newSeqSetup(t)
	require.NoError(t, s.seq.SetMaxSafeLag(context.Background(), 1))

	// Safe head is exactly at the bound: building the next block will exceed it.
	deliver(s.seq, engine.ForkchoiceUpdateEvent{
		UnsafeL2Head: s.head,
		SafeL2Head:   eth.L2BlockRef{Hash: common.Hash{0x30}, Number: s.head.Number},
	})
	_, ok := s.seq.NextAction()
	require.True(t, ok, "not stalled yet at the bound")

	// Build, seal, and insert one block via direct calls.
	info := s.startBuild(t)
	envelope, ref := s.sealedPayload()
	s.deps.eng.sealBuildFn = func(ctx context.Context, gotInfo eth.PayloadInfo, buildStarted time.Time) (*engine.SealResult, error) {
		require.Equal(t, info, gotInfo)
		return &engine.SealResult{Envelope: envelope, Ref: ref}, nil
	}
	s.deps.eng.processPayloadFn = func(ctx context.Context, envelope *eth.ExecutionPayloadEnvelope, ref eth.L2BlockRef, buildStarted time.Time) error {
		return nil
	}
	s.seq.RunAction()
	require.Equal(t, ref, s.seq.unsafeHead, "block inserted via direct path")

	// The insertion armed the next block, but the safe head (as of the last
	// ingested forkchoice update) now lags beyond the bound: stalled.
	_, ok = s.seq.NextAction()
	require.False(t, ok, "stalled on the direct path, before any forkchoice echo")

	// The safe head catches up: resume.
	deliver(s.seq, engine.ForkchoiceUpdateEvent{
		UnsafeL2Head: ref,
		SafeL2Head:   eth.L2BlockRef{Hash: common.Hash{0x31}, Number: ref.Number},
	})
	_, ok = s.seq.NextAction()
	require.True(t, ok, "resumed after safe head caught up")
}

// TestSequencerActionHonorsParkingDrain checks the replay-to-action boundary:
// when an action fires, RunAction first replays the inbox; if that replay
// parks the schedule (e.g. a reset arrived while the timer was pending), the
// action must not run against the pre-drain decision.
func TestSequencerActionHonorsParkingDrain(t *testing.T) {
	s := newSeqSetup(t)
	_, ok := s.seq.NextAction()
	require.True(t, ok, "armed before the reset arrives")

	// The reset is appended to the inbox but not yet drained, as if it arrived
	// while the action timer was already firing.
	s.seq.OnEvent(context.Background(), rollup.ResetEvent{Err: errors.New("mock reset")})

	s.deps.eng.startBuildFn = func(ctx context.Context, attrs *derive.AttributesWithParent) (*engine.BuildStartResult, error) {
		t.Fatal("must not start building during a reset")
		return nil, nil
	}
	s.seq.RunAction()
	_, ok = s.seq.NextAction()
	require.False(t, ok, "parked by the replayed reset")
	require.Equal(t, BuildingState{}, s.seq.building, "no build started")
}

// TestSequencerStopAfterStaleProcess checks that Stop does not wait for a
// sealed-and-dropped block to become the head: a stale ProcessPayload resets
// the sealed marker, so a leader can stop promptly after losing the race.
func TestSequencerStopAfterStaleProcess(t *testing.T) {
	s := newSeqSetup(t)
	info := s.startBuild(t)
	envelope, ref := s.sealedPayload()
	s.deps.eng.sealBuildFn = func(ctx context.Context, gotInfo eth.PayloadInfo, buildStarted time.Time) (*engine.SealResult, error) {
		require.Equal(t, info, gotInfo)
		return &engine.SealResult{Envelope: envelope, Ref: ref}, nil
	}
	s.deps.eng.processPayloadFn = func(ctx context.Context, envelope *eth.ExecutionPayloadEnvelope, ref eth.L2BlockRef, buildStarted time.Time) error {
		return engine.ErrStaleBuild // competing block won after we gossiped
	}
	s.seq.RunAction()
	require.Equal(t, s.seq.unsafeHead, s.seq.lastSealed, "nothing outstanding after the stale drop")

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	hash, err := s.seq.Stop(ctx)
	require.NoError(t, err, "Stop must not wait for the dropped block")
	require.Equal(t, s.seq.unsafeHead.Hash, hash)
}

// TestSequencerRearmsAfterStaleBufferedPayload covers the retry of a payload
// left in the async-gossip buffer by an earlier temporary insertion failure.
// If the chain has already moved on by the time it is retried, the engine
// rejects it and requests a forkchoice update — but that update names the head
// the sequencer already knows, so it re-plans nothing. Waiting for it would
// park an active leader forever.
func TestSequencerRearmsAfterStaleBufferedPayload(t *testing.T) {
	s := newSeqSetup(t)
	envelope, ref := s.sealedPayload()

	// A previous action sealed and gossiped this payload, then failed to insert
	// it with a temporary error, so it stayed in the buffer.
	s.deps.asyncGossip.payload = envelope
	s.seq.lastSealed = ref

	// Meanwhile a competing block at the same height became the head, and the
	// sequencer has already ingested that update.
	competing := eth.L2BlockRef{
		Hash:       common.Hash{0xc1},
		Number:     ref.Number,
		ParentHash: s.head.Hash,
		Time:       ref.Time,
		L1Origin:   s.head.L1Origin,
	}
	deliver(s.seq, engine.ForkchoiceUpdateEvent{UnsafeL2Head: competing})
	require.Equal(t, competing, s.seq.unsafeHead)
	_, ok := s.seq.NextAction()
	require.True(t, ok, "the competing head re-armed the sequencer")

	// Retrying the buffered payload now fails: it does not extend the new head.
	s.deps.eng.processPayloadFn = func(context.Context, *eth.ExecutionPayloadEnvelope, eth.L2BlockRef, time.Time) error {
		return engine.ErrStaleBuild
	}
	s.seq.RunAction()

	require.Nil(t, s.deps.asyncGossip.payload, "the stale payload is discarded")
	next, ok := s.seq.NextAction()
	require.True(t, ok, "must re-plan locally; the requested forkchoice update is one we already have")
	require.Equal(t, competing, s.seq.unsafeHead, "the next build targets the current head")
	require.False(t, next.After(time.Unix(int64(competing.Time+s.deps.cfg.BlockTime), 0)),
		"scheduled no later than the next block's slot")
}

// TestSequencerStopAfterRejectedInsert checks that a block discarded as
// invalid after it was sealed does not leave Stop waiting for a head that can
// never arrive: handleInvalid reconciles the sealed marker.
func TestSequencerStopAfterRejectedInsert(t *testing.T) {
	s := newSeqSetup(t)
	info := s.startBuild(t)
	envelope, ref := s.sealedPayload()
	s.deps.eng.sealBuildFn = func(ctx context.Context, gotInfo eth.PayloadInfo, buildStarted time.Time) (*engine.SealResult, error) {
		require.Equal(t, info, gotInfo)
		return &engine.SealResult{Envelope: envelope, Ref: ref}, nil
	}
	s.deps.eng.processPayloadFn = func(ctx context.Context, envelope *eth.ExecutionPayloadEnvelope, ref eth.L2BlockRef, buildStarted time.Time) error {
		return engine.ErrPayloadInvalid
	}
	s.seq.RunAction()
	require.Equal(t, s.seq.unsafeHead, s.seq.lastSealed, "rejected block is not left outstanding")

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	hash, err := s.seq.Stop(ctx)
	require.NoError(t, err, "Stop must not wait for the rejected block")
	require.Equal(t, s.seq.unsafeHead.Hash, hash)
}

// TestSequencerRearmsOnRewoundHead reproduces a stall seen in the flashblocks
// acceptance tests: the engine head is rewound (reset, block replacement,
// backup-unsafe restore) while a block is being built, so the engine rejects
// the seal as stale and the sequencer parks. The forkchoice update that
// recovers it names the *rewound*, lower-numbered head, so re-planning only on
// a higher block number leaves the sequencer parked forever.
func TestSequencerRearmsOnRewoundHead(t *testing.T) {
	s := newSeqSetup(t)
	info := s.startBuild(t)
	_, ref := s.sealedPayload()

	// The chain moved off our parent while we were building, so the engine
	// refuses to hand us the sealed block.
	s.deps.eng.sealBuildFn = func(ctx context.Context, gotInfo eth.PayloadInfo, buildStarted time.Time) (*engine.SealResult, error) {
		require.Equal(t, info, gotInfo)
		return nil, engine.ErrStaleBuild
	}
	s.seq.RunAction()
	require.Equal(t, BuildingState{}, s.seq.building, "stale job dropped")
	_, ok := s.seq.NextAction()
	require.False(t, ok, "parked until the engine reports its head")

	// The engine's requested forkchoice update carries the rewound head, one
	// block *below* what we had recorded.
	rewound := eth.L2BlockRef{
		Hash:     common.Hash{0x55},
		Number:   s.head.Number - 1,
		L1Origin: s.head.L1Origin,
		Time:     s.head.Time - s.deps.cfg.BlockTime,
	}
	require.Less(t, rewound.Number, s.seq.unsafeHead.Number, "the recovering update rewinds the head")
	deliver(s.seq, engine.ForkchoiceUpdateEvent{UnsafeL2Head: rewound})

	require.Equal(t, rewound, s.seq.unsafeHead, "the rewound head is adopted")
	_, ok = s.seq.NextAction()
	require.True(t, ok, "a rewind re-arms the sequencer instead of parking it forever")
	require.NotEqual(t, ref.Hash, s.seq.unsafeHead.Hash)
}

// TestSequencerSetMaxSafeLagResumes checks that relaxing the bound at runtime
// resumes a stalled sequencer immediately: while stalled its goroutine is
// parked with no timer armed, so nothing else would re-evaluate the bound.
func TestSequencerSetMaxSafeLagResumes(t *testing.T) {
	s := newSeqSetup(t)
	require.NoError(t, s.seq.SetMaxSafeLag(context.Background(), 2))
	deliver(s.seq, engine.ForkchoiceUpdateEvent{
		UnsafeL2Head: eth.L2BlockRef{Hash: common.Hash{0x31}, Number: s.head.Number + 1, Time: s.head.Time + 2},
		SafeL2Head:   eth.L2BlockRef{Hash: common.Hash{0x30}, Number: s.head.Number - 1},
	})
	_, ok := s.seq.NextAction()
	require.False(t, ok, "stalled by max safe lag")
	<-s.seq.wakeCh // drain the wake from the ingest above

	require.NoError(t, s.seq.SetMaxSafeLag(context.Background(), 0))
	next, ok := s.seq.NextAction()
	require.True(t, ok, "disabling the bound resumes sequencing without a forkchoice update")
	require.Equal(t, s.clock.Now(), next, "resume immediately")
	require.Len(t, s.seq.wakeCh, 1, "the parked goroutine is woken to re-plan")
}

type sequencerTestDeps struct {
	cfg              *rollup.Config
	attribBuilder    *FakeAttributesBuilder
	l1OriginSelector *FakeL1OriginSelector
	seqState         *BasicSequencerStateListener
	conductor        *FakeConductor
	asyncGossip      *FakeAsyncGossip
	eng              *fakeEngController
}

func createSequencer(log log.Logger) (*Sequencer, *sequencerTestDeps) {
	rng := rand.New(rand.NewSource(123))
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
	eng := &fakeEngController{}
	deps := &sequencerTestDeps{
		cfg:           cfg,
		attribBuilder: &FakeAttributesBuilder{cfg: cfg, rng: rng},
		l1OriginSelector: &FakeL1OriginSelector{
			l1OriginFn: func(l2Head eth.L2BlockRef) (eth.L1BlockRef, error) {
				panic("override this")
			},
		},
		seqState:    &BasicSequencerStateListener{},
		conductor:   &FakeConductor{},
		asyncGossip: &FakeAsyncGossip{},
		eng:         eng,
	}
	seq := NewSequencer(context.Background(), log, cfg, defaultSealingDuration, deps.attribBuilder,
		deps.l1OriginSelector, deps.seqState, deps.conductor,
		deps.asyncGossip, metrics.NoopMetrics, eng)
	// We create mock payloads, with the epoch-id as tx[0], rather than proper L1Block-info deposit tx.
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
	return seq, deps
}

// TestSequencerStaysParkedUntilResetConfirmed pins the reset ordering: the
// engine emits the rewound forkchoice update before EngineResetConfirmedEvent.
// If that update re-armed the sequencer, the confirmation's deliberate
// one-block cool-down — which keeps a reset loop from running hot — would be
// skipped, and the rewound head's past timestamp would schedule immediately.
func TestSequencerStaysParkedUntilResetConfirmed(t *testing.T) {
	s := newSeqSetup(t)

	deliver(s.seq, rollup.ResetEvent{Err: errors.New("mock reset")})
	_, ok := s.seq.NextAction()
	require.False(t, ok, "reset parks the sequencer")

	rewound := s.head
	rewound.Hash = common.Hash{0x33}
	rewound.Number = s.head.Number - 1
	rewound.Time = s.head.Time - s.deps.cfg.BlockTime
	deliver(s.seq, engine.ForkchoiceUpdateEvent{UnsafeL2Head: rewound})
	_, ok = s.seq.NextAction()
	require.False(t, ok, "the pre-confirmation head update must not re-arm the sequencer")
	require.Equal(t, rewound, s.seq.unsafeHead,
		"the head is still recorded, so the confirmation arms onto the rewound head")

	deliver(s.seq, engine.EngineResetConfirmedEvent{})
	next, ok := s.seq.NextAction()
	require.True(t, ok, "the confirmation resumes sequencing")
	require.Equal(t, s.clock.Now().Add(s.blockTime()), next,
		"the confirmation applies the one-block cool-down")
}

// TestSequencerMaxSafeLagHoldsThroughRecovery covers the two re-arm paths that
// can release a maxSafeLag stall: an engine temporary error, and a reset
// confirmation. Both must re-check the bound instead of arming unconditionally,
// or the sequencer builds a block past a bound that is still exceeded.
func TestSequencerMaxSafeLagHoldsThroughRecovery(t *testing.T) {
	stall := func(t *testing.T) *seqTestSetup {
		s := newSeqSetup(t)
		require.NoError(t, s.seq.SetMaxSafeLag(context.Background(), 1))
		ahead := s.head
		ahead.Hash = common.Hash{0x44}
		ahead.Number = s.head.Number + 5
		deliver(s.seq, engine.ForkchoiceUpdateEvent{UnsafeL2Head: ahead, SafeL2Head: s.head})
		_, ok := s.seq.NextAction()
		require.False(t, ok, "unsafe head past the lag bound stalls the sequencer")
		return s
	}

	t.Run("engine temporary error", func(t *testing.T) {
		s := stall(t)
		deliver(s.seq, rollup.EngineTemporaryErrorEvent{Err: errors.New("mock temp error")})
		_, ok := s.seq.NextAction()
		require.False(t, ok, "a temporary-error backoff must not release the stall")
	})

	t.Run("reset confirmation", func(t *testing.T) {
		s := stall(t)
		deliver(s.seq, rollup.ResetEvent{Err: errors.New("mock reset")})
		deliver(s.seq, engine.EngineResetConfirmedEvent{})
		_, ok := s.seq.NextAction()
		require.False(t, ok, "a reset confirmation must not release the stall")
	})
}

// TestSequencerPayloadSuccessClearsGossip covers the retained ingest path for
// derivation-originated payloads: the async-gossip buffer must be cleared when
// the inserted block is the one we were building, so a stale payload cannot be
// reused, and left alone otherwise.
func TestSequencerPayloadSuccessClearsGossip(t *testing.T) {
	s := newSeqSetup(t)
	envelope, ref := s.sealedPayload()

	s.seq.building = BuildingState{Ref: ref}
	s.deps.asyncGossip.payload = envelope

	unrelated := &eth.ExecutionPayloadEnvelope{ExecutionPayload: &eth.ExecutionPayload{
		BlockHash: common.Hash{0xaa},
	}}
	deliver(s.seq, engine.PayloadSuccessEvent{Envelope: unrelated})
	require.NotNil(t, s.deps.asyncGossip.payload, "another block's insertion is not ours to act on")
	require.Equal(t, ref, s.seq.building.Ref, "build state untouched")

	deliver(s.seq, engine.PayloadSuccessEvent{Envelope: envelope})
	require.Nil(t, s.deps.asyncGossip.payload, "our block was inserted: drop the gossip buffer")
	require.Equal(t, BuildingState{}, s.seq.building, "our block was inserted: build state is done")
}

// TestSequencerStaysParkedOnTemporaryErrorDuringReset covers the window between
// a reset signal and its confirmation, during which the engine has not rewound
// anything yet: an engine temporary error must not re-arm the sequencer, or it
// resumes building on the pre-reset head — a chain it has already decided must
// be rewound — and can seal and gossip a block there.
func TestSequencerStaysParkedOnTemporaryErrorDuringReset(t *testing.T) {
	s := newSeqSetup(t)

	deliver(s.seq, rollup.ResetEvent{Err: errors.New("mock reset")})
	_, ok := s.seq.NextAction()
	require.False(t, ok, "reset parks the sequencer")

	// The engine is still resolving the new heads (FindL2Heads); ordinary
	// temporary errors keep arriving in the meantime.
	deliver(s.seq, rollup.EngineTemporaryErrorEvent{Err: errors.New("mock temporary error")})
	_, ok = s.seq.NextAction()
	require.False(t, ok, "an error backoff must not pre-empt the reset confirmation")

	// Even forced, no build may start on the head the reset is about to discard.
	s.deps.eng.startBuildFn = func(context.Context, *derive.AttributesWithParent) (*engine.BuildStartResult, error) {
		t.Fatal("must not start a build on the pre-reset head while the reset is unconfirmed")
		return nil, nil
	}
	s.seq.RunAction()

	deliver(s.seq, engine.EngineResetConfirmedEvent{})
	next, ok := s.seq.NextAction()
	require.True(t, ok, "the confirmation still resumes sequencing")
	require.Equal(t, s.clock.Now().Add(s.blockTime()), next, "with its cool-down intact")
}

// TestSequencerStartDuringResetStaysParked is the same rule for the operator
// path. Start only checks that the caller's head matches unsafeHead, which
// during an unconfirmed reset is still the pre-reset head — so a conductor
// promoting a mid-reset node succeeds, and must not sequence on the chain the
// reset is about to discard.
func TestSequencerStartDuringResetStaysParked(t *testing.T) {
	s := newSeqSetup(t)
	// Stop waits for the head to reach what we last sealed; nothing is in flight.
	s.seq.lastSealed = s.head
	_, err := s.seq.Stop(context.Background())
	require.NoError(t, err)

	deliver(s.seq, rollup.ResetEvent{Err: errors.New("mock reset")})
	require.NoError(t, s.seq.Start(context.Background(), s.head.Hash),
		"the pre-reset head still matches, so Start succeeds")
	require.True(t, s.seq.Active(), "the sequencer is started")
	_, ok := s.seq.NextAction()
	require.False(t, ok, "but stays parked until the reset is confirmed")

	deliver(s.seq, engine.EngineResetConfirmedEvent{})
	_, ok = s.seq.NextAction()
	require.True(t, ok, "the confirmation arms it on the post-reset head")
}
