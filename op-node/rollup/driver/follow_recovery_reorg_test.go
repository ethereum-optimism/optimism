package driver

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-node/metrics"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/engine"
	syncconfig "github.com/ethereum-optimism/optimism/op-node/rollup/sync"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/event"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
)

// Characterizes an already-resolved claim withdrawal. All private ancestry and
// L1 origins remain unchanged. This does not simulate the full L1 publication
// reorg, but proves that the lower recovery anchor alone rewinds execution.
func TestFollowRecoveryCheckpointRetreatRewindsUnsafe(t *testing.T) {
	ctx := t.Context()
	l1 := eth.L1BlockRef{Hash: common.Hash{0xa0}, Number: 100, Time: 1000}
	l2 := &recoveryBranchL2{prefixL2: prefixL2{refs: map[common.Hash]eth.L2BlockRef{}}, canonical: map[uint64]eth.L2BlockRef{}}
	chain := make([]eth.L2BlockRef, 11)
	for n := range chain {
		chain[n] = eth.L2BlockRef{Hash: common.Hash{byte(n + 1)}, Number: uint64(n), Time: 1000 + uint64(n)*2,
			L1Origin: l1.ID(), SequenceNumber: uint64(n)}
		if n > 0 {
			chain[n].ParentHash = chain[n-1].Hash
		}
		l2.refs[chain[n].Hash], l2.canonical[uint64(n)] = chain[n], chain[n]
	}
	genesis, anchor, oldSafe, oldUnsafe := chain[0], chain[4], chain[8], chain[10]
	el := &testutils.MockEngine{}
	em := event.EmitterFunc(func(context.Context, event.Event) {})
	ec := engine.NewEngineController(ctx, el, testlog.Logger(t, log.LevelError), metrics.NoopMetrics,
		&rollup.Config{}, &syncconfig.Config{L2FollowSourceEndpoint: "http://localhost"}, &testutils.MockL1Source{}, em, nil)
	ec.SetUnsafeHead(oldUnsafe)
	ec.SetLocalSafeHead(oldSafe)
	ec.SetPendingSafeL2Head(oldSafe)
	//nolint:staticcheck // Follow mode stores externally supplied cross-safe in this field.
	ec.SetDeprecatedSafeHead(oldSafe)
	ec.SetFinalizedHead(genesis)
	// Pin the actual outgoing forkchoice carrying the lower unsafe head.
	el.ExpectForkchoiceUpdate(&eth.ForkchoiceState{HeadBlockHash: anchor.Hash, SafeBlockHash: anchor.Hash, FinalizedBlockHash: genesis.Hash}, nil,
		&eth.ForkchoiceUpdatedResult{PayloadStatus: eth.PayloadStatusV1{Status: eth.ExecutionValid}}, nil)
	el.ExpectL2BlockRefByNumber(anchor.Number, anchor, nil)
	paused := false
	f := &followRecovery{enabled: true, l2: l2, engine: ec, emitter: em,
		pause:  func(p bool) { paused = paused || p },
		status: &sources.FollowStatus{Recovery: &sources.FollowRecoveryStatus{Anchor: oldSafe, Target: oldSafe}}}
	publicAnchor := anchor
	publicAnchor.Hash = common.Hash{0xf4}
	status := &sources.FollowStatus{LocalSafeL2: anchor, SafeL2: anchor, FinalizedL2: genesis, CurrentL1: l1,
		Recovery: &sources.FollowRecoveryStatus{Anchor: anchor, Target: publicAnchor, Safe: publicAnchor, Finalized: genesis}}
	require.NoError(t, f.update(ctx, status))
	require.True(t, paused)
	require.Equal(t, anchor, ec.UnsafeL2Head())
	require.Equal(t, anchor, ec.LocalSafeHead())
	require.Equal(t, oldUnsafe, l2.canonical[oldUnsafe.Number], "fixture never changed the private branch or its L1 inputs")
	el.AssertExpectations(t)
}

// The source combines remote follow status with the local L1 view, as the node
// wiring does. A conflicting L1 origin rejects the snapshot; it does not emit
// a reset itself (this Driver has no engine/reset machinery attached).
func TestFollowUpstreamRejectsNonCanonicalL1Origin(t *testing.T) {
	source := &blockingUpstreamFollowSource{
		status: &sources.FollowStatus{
			LocalSafeL2: eth.L2BlockRef{Number: 8, L1Origin: eth.BlockID{Number: 1, Hash: common.Hash{0xff}}},
			CurrentL1:   eth.L1BlockRef{Number: 2, Hash: common.Hash{2}},
		}, started: make(chan struct{}, 1), release: make(chan struct{}),
	}
	close(source.release)
	m := &followMetricsStub{}
	d := &Driver{StatusTracker: &followStatusTrackerStub{}, driverCtx: t.Context(), upstreamFollowSource: source, metrics: m, log: testlog.Logger(t, log.LevelError)}
	status := d.followUpstream()
	require.Nil(t, status)
	require.Equal(t, []uint64{1}, source.calls)
	require.Equal(t, []string{"error_l1_mismatch"}, m.results)
}
