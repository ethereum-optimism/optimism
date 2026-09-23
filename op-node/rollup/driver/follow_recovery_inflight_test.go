package driver

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/ethereum-optimism/optimism/op-node/metrics"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/engine"
	syncconfig "github.com/ethereum-optimism/optimism/op-node/rollup/sync"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/event"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestFollowRecoveryRejectsSupersededCompletion(t *testing.T) {
	for _, scenario := range []string{"plan changes before completion", "reorg after fetch", "source unavailable after fetch"} {
		t.Run(scenario, func(t *testing.T) {
			ctx := t.Context()
			genesis := eth.L2BlockRef{Hash: common.Hash{1}, Time: 100}
			retained := eth.L2BlockRef{Hash: common.Hash{2}, ParentHash: genesis.Hash, Number: 1, Time: 102, SequenceNumber: 1}
			stale := eth.L2BlockRef{Hash: common.Hash{3}, ParentHash: retained.Hash, Number: 2, Time: 104, SequenceNumber: 2}
			publicRetained, publicOld := retained, stale
			publicRetained.Hash, publicOld.Hash = common.Hash{12}, common.Hash{13}
			publicNew := publicOld
			publicNew.Hash = common.Hash{14}
			publicNew.L1Origin = eth.BlockID{Hash: common.Hash{21}, Number: 1}
			publicNew.SequenceNumber = 0
			l2 := &recoveryBranchL2{prefixL2: prefixL2{refs: map[common.Hash]eth.L2BlockRef{genesis.Hash: genesis}},
				canonical: map[uint64]eth.L2BlockRef{0: genesis, 1: retained, 2: stale}}
			el := &testutils.MockEngine{}
			el.On("ForkchoiceUpdate", mock.Anything, mock.Anything).Return(&eth.ForkchoiceUpdatedResult{PayloadStatus: eth.PayloadStatusV1{Status: eth.ExecutionValid}}, nil).Maybe()
			var noError error
			el.On("L2BlockRefByNumber", retained.Number).Return(retained, &noError).Maybe()
			el.On("L2BlockRefByNumber", stale.Number).Return(stale, &noError).Maybe()
			em := event.EmitterFunc(func(context.Context, event.Event) {})
			ec := engine.NewEngineController(ctx, el, testlog.Logger(t, 0), metrics.NoopMetrics, &rollup.Config{},
				&syncconfig.Config{L2FollowSourceEndpoint: "http://localhost"}, &testutils.MockL1Source{}, em, nil)
			ec.SetUnsafeHead(retained)
			ec.SetLocalSafeHead(retained)
			ec.SetPendingSafeL2Head(retained)
			ec.SetFinalizedHead(genesis)
			//nolint:staticcheck // Follow mode stores externally supplied cross-safe here.
			ec.SetDeprecatedSafeHead(retained)
			status := func(target eth.L2BlockRef) *sources.FollowStatus {
				return &sources.FollowStatus{LocalSafeL2: genesis, SafeL2: genesis, FinalizedL2: genesis,
					CurrentL1: eth.L1BlockRef{Hash: common.Hash{20}, Number: 20},
					Recovery:  &sources.FollowRecoveryStatus{Anchor: genesis, Target: target, Safe: target, Finalized: genesis}}
			}
			journal := &recoveryJournal{path: filepath.Join(t.TempDir(), "recovery.db"), genesis: genesis.Hash}
			require.NoError(t, journal.commitProgress(0, &recoveryProgress{Anchor: genesis, Public: publicRetained, Private: retained}))
			paused, unavailable := true, false
			canonical := publicOld
			f := &followRecovery{enabled: true, l2: l2, engine: ec, emitter: em, journal: journal,
				pause: func(value bool) { paused = value }, status: status(publicOld), applied: publicRetained, appliedPrivate: retained,
				source: recoverySourceFunc(func(_ context.Context, n uint64, target eth.BlockID) (eth.L2BlockRef, error) {
					if unavailable || target != canonical.ID() {
						return eth.L2BlockRef{}, errors.New("recovery snapshot revoked")
					}
					if n == retained.Number {
						return publicRetained, nil
					}
					require.Equal(t, canonical.Number, n)
					return canonical, nil
				}),
				builder: recoveryBuilderFunc(func(context.Context, eth.L2BlockRef, eth.BlockID) (*eth.PayloadAttributes, error) {
					return &eth.PayloadAttributes{Timestamp: 104, NoTxPool: true, Transactions: []eth.Data{{0x7e}}}, nil
				})}
			require.NoError(t, f.next(ctx, retained))
			require.NotNil(t, f.build)
			// The engine executed the fetched attributes, but its completion event
			// has not reached the follower when the public branch changes.
			ec.SetUnsafeHead(stale)
			ec.SetLocalSafeHead(stale)
			canonical = publicNew
			if scenario == "plan changes before completion" {
				require.NoError(t, f.update(ctx, status(publicNew)))
			} else if scenario == "source unavailable after fetch" {
				unavailable = true
			}
			require.True(t, f.OnEvent(ctx, engine.LocalSafeUpdateEvent{Ref: stale}))
			require.True(t, paused, "obsolete completion must not resume sequencing")
			require.Equal(t, retained, ec.SafeL2Head(), "obsolete completion must not advance cross-safe")
			require.Equal(t, publicRetained, journal.progress.Public, "obsolete completion must not be persisted")
			unavailable = false
			require.NoError(t, f.update(ctx, status(publicNew)))
			require.Equal(t, retained, ec.UnsafeL2Head(), "discard the obsolete in-flight block even when the prior checkpoint survives")
			require.Equal(t, retained, ec.LocalSafeHead())
			require.Nil(t, f.build)
			require.NoError(t, f.next(ctx, retained))
			require.Equal(t, publicNew, f.build.public, "resume against the new canonical recovery schedule")
			require.True(t, f.OnEvent(ctx, engine.LocalSafeUpdateEvent{Ref: stale}))
			require.True(t, paused, "a late old completion cannot finish the new build")
			require.Equal(t, publicRetained, journal.progress.Public)
			require.NotNil(t, f.build)
		})
	}
}
