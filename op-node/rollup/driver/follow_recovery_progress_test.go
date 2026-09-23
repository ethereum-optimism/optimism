package driver

import (
	"context"
	"errors"
	"os"
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

func TestCompletedPrivateRecoveryRestartPreservesUnsafeSuffix(t *testing.T) {
	for _, scenario := range []string{"completed", "empty progression", "partial replay", "public reorg", "private reorg", "new prefix", "target retreat", "source unavailable", "write failure"} {
		t.Run(scenario, func(t *testing.T) {
			ctx := t.Context()
			l2 := &recoveryBranchL2{prefixL2: prefixL2{refs: make(map[common.Hash]eth.L2BlockRef)}, canonical: make(map[uint64]eth.L2BlockRef)}
			original := make([]eth.L2BlockRef, 8)
			for n := range original {
				ref := eth.L2BlockRef{Hash: common.Hash{byte(n + 1)}, Number: uint64(n), Time: uint64(100 + 2*n), SequenceNumber: uint64(n)}
				if n > 0 {
					ref.ParentHash = original[n-1].Hash
				}
				original[n] = ref
				l2.refs[ref.Hash], l2.canonical[ref.Number] = ref, ref
			}
			base := original[0]
			prefix := &sources.FollowRecoveryPrefix{Parent: original[7].ID(), Last: original[4]}
			prefix.Last.Hash = common.Hash{0xff}
			path := filepath.Join(t.TempDir(), "recovery.db")
			journal := &recoveryJournal{path: path, genesis: base.Hash}
			f := &followRecovery{l2: l2, journal: journal}
			_, err := f.prefixAnchor(ctx, base, prefix)
			require.NoError(t, err)
			require.NoError(t, journal.commit(0))
			// Recovery replaces the rejected suffix, then ordinary sequencing adds two
			// newer valid blocks. These must survive a restart after replay completes.
			for n := uint64(5); n <= 10; n++ {
				ref := eth.L2BlockRef{Hash: common.Hash{byte(n + 100)}, ParentHash: l2.canonical[n-1].Hash, Number: n, Time: 100 + 2*n, SequenceNumber: n}
				l2.refs[ref.Hash], l2.canonical[n] = ref, ref
			}
			replacement, suffix := l2.canonical[8], l2.canonical[10]
			if scenario == "partial replay" {
				replacement = l2.canonical[6]
			}
			public := replacement
			public.Hash, public.ParentHash = common.Hash{200}, common.Hash{199}
			status := &sources.FollowStatus{LocalSafeL2: base, SafeL2: base, FinalizedL2: base, CurrentL1: eth.L1BlockRef{Hash: common.Hash{9}, Number: 20}, Recovery: &sources.FollowRecoveryStatus{Anchor: base, Target: public, Safe: base, Finalized: base, Prefix: prefix}}
			if scenario == "empty progression" {
				status.Recovery.Prefix = nil
			}
			paused := false
			sourceUnavailable := false
			sourcePublic := public
			newFollower := func(unsafe eth.L2BlockRef) *followRecovery {
				el := &testutils.MockEngine{}
				var noError error
				for n, ref := range l2.canonical {
					el.On("L2BlockRefByNumber", n).Return(ref, &noError).Maybe()
				}
				el.On("ForkchoiceUpdate", mock.Anything, mock.Anything).Return(&eth.ForkchoiceUpdatedResult{PayloadStatus: eth.PayloadStatusV1{Status: eth.ExecutionValid}}, nil).Maybe()
				em := event.EmitterFunc(func(context.Context, event.Event) {})
				ec := engine.NewEngineController(ctx, el, testlog.Logger(t, 0), metrics.NoopMetrics, &rollup.Config{}, &syncconfig.Config{L2FollowSourceEndpoint: "http://localhost"}, &testutils.MockL1Source{}, em, nil)
				ec.SetUnsafeHead(unsafe)
				ec.SetLocalSafeHead(replacement)
				ec.SetFinalizedHead(base)
				//nolint:staticcheck // Follow mode stores externally supplied cross-safe here.
				ec.SetDeprecatedSafeHead(base)
				return &followRecovery{l2: l2, engine: ec, emitter: em, pause: func(v bool) { paused = v }, enabled: true, journal: &recoveryJournal{path: path, genesis: base.Hash}, source: recoverySourceFunc(func(_ context.Context, n uint64, target eth.BlockID) (eth.L2BlockRef, error) {
					require.Equal(t, public.Number, n)
					require.Equal(t, public.ID(), target)
					if sourceUnavailable {
						return eth.L2BlockRef{}, errors.New("source unavailable")
					}
					return sourcePublic, nil
				})}
			}
			f = newFollower(replacement)
			if scenario == "write failure" {
				occupied := filepath.Join(t.TempDir(), "occupied")
				require.NoError(t, os.WriteFile(occupied, []byte("occupied"), 0600))
				f.journal.path = filepath.Join(occupied, "recovery.db")
			}
			f.status, f.build = status, &recoveryBuild{public: public, parent: replacement.ParentHash}
			require.True(t, f.OnEvent(ctx, engine.LocalSafeUpdateEvent{Ref: replacement}))
			if scenario == "write failure" {
				require.True(t, paused, "failed persistence must not resume sequencing")
				f.journal.path = path
				require.NoError(t, f.update(ctx, status), "status updates retry failed persistence")
				require.False(t, paused)
			}
			require.Equal(t, scenario != "partial replay", f.canResume(replacement.Number))
			expectedUnsafe, expectedLocal := suffix, replacement
			switch scenario {
			case "partial replay":
				expectedUnsafe = replacement
			case "public reorg":
				sourcePublic.Hash = common.Hash{201}
				expectedUnsafe, expectedLocal = original[4], original[4]
			case "private reorg":
				fork := replacement
				fork.Hash = common.Hash{202}
				l2.canonical[8] = fork
				expectedUnsafe, expectedLocal = original[4], original[4]
			case "new prefix":
				newPrefix := *prefix
				newPrefix.Last = original[3]
				status.Recovery.Prefix = &newPrefix
				expectedUnsafe, expectedLocal = original[3], original[3]
			case "target retreat":
				status.Recovery.Target.Number = 7
				expectedUnsafe, expectedLocal = original[4], original[4]
			case "source unavailable":
				sourceUnavailable = true
			}
			restarted := newFollower(suffix)
			err = restarted.update(ctx, status)
			if scenario == "source unavailable" {
				require.ErrorContains(t, err, "source unavailable")
				require.True(t, paused)
				require.Equal(t, expectedUnsafe, restarted.engine.UnsafeL2Head())
				return
			}
			require.NoError(t, err)
			require.Equal(t, expectedUnsafe, restarted.engine.UnsafeL2Head(), "completed recovery must not discard newer valid private history")
			require.Equal(t, expectedLocal, restarted.engine.LocalSafeHead())
			if expectedLocal == replacement {
				require.Equal(t, public, restarted.applied)
			} else {
				require.Zero(t, restarted.applied)
				require.Nil(t, restarted.journal.progress, "revoked replay progress must be durably cleared")
				reopened := &recoveryJournal{path: path, genesis: base.Hash}
				require.NoError(t, reopened.load())
				require.Nil(t, reopened.progress)
			}
		})
	}
}
