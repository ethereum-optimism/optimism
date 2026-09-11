package driver

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/ethereum-optimism/optimism/op-node/metrics"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
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

type recoverySourceFunc func(context.Context, uint64, eth.BlockID) (eth.L2BlockRef, error)

func (fn recoverySourceFunc) RecoveryBlock(ctx context.Context, n uint64, target eth.BlockID) (eth.L2BlockRef, error) {
	return fn(ctx, n, target)
}

type recoveryBuilderFunc func(context.Context, eth.L2BlockRef, eth.BlockID) (*eth.PayloadAttributes, error)

func (fn recoveryBuilderFunc) PreparePayloadAttributes(ctx context.Context, parent eth.L2BlockRef, origin eth.BlockID) (*eth.PayloadAttributes, error) {
	return fn(ctx, parent, origin)
}

func TestFollowRecoveryAttributes(t *testing.T) {
	for _, scenario := range []string{"valid", "source unavailable", "wrong timestamp", "wrong sequence", "txpool enabled"} {
		t.Run(scenario, func(t *testing.T) {
			parent := eth.L2BlockRef{Hash: common.Hash{1}, Number: 8, Time: 100, L1Origin: eth.BlockID{Hash: common.Hash{2}, Number: 20}, SequenceNumber: 3}
			public := eth.L2BlockRef{Hash: common.Hash{3}, ParentHash: common.Hash{4}, Number: 9, Time: 102, L1Origin: parent.L1Origin, SequenceNumber: 4}
			frontier := eth.L1BlockRef{Hash: common.Hash{5}, Number: 35}
			emitter := &testutils.MockEmitter{}
			ec := engine.NewEngineController(t.Context(), &testutils.MockEngine{}, testlog.Logger(t, 0), metrics.NoopMetrics,
				&rollup.Config{}, &syncconfig.Config{}, &testutils.MockL1Source{}, emitter, nil)
			ec.SetLocalSafeHead(parent)
			f := &followRecovery{engine: ec, emitter: emitter, status: &sources.FollowStatus{
				CurrentL1: frontier, Recovery: &sources.FollowRecoveryStatus{Anchor: parent, Target: public},
			}}
			f.source = recoverySourceFunc(func(_ context.Context, n uint64, target eth.BlockID) (eth.L2BlockRef, error) {
				require.Equal(t, uint64(9), n)
				require.Equal(t, public.ID(), target)
				if scenario == "source unavailable" {
					return eth.L2BlockRef{}, errors.New("unavailable")
				}
				ref := public
				if scenario == "wrong sequence" {
					ref.SequenceNumber++
				}
				return ref, nil
			})
			f.builder = recoveryBuilderFunc(func(_ context.Context, onto eth.L2BlockRef, origin eth.BlockID) (*eth.PayloadAttributes, error) {
				require.Equal(t, parent, onto, "execute against the private parent")
				require.Equal(t, public.L1Origin, origin)
				attrs := &eth.PayloadAttributes{Timestamp: 102, NoTxPool: true, Transactions: []eth.Data{{0x7e}}}
				if scenario == "wrong timestamp" {
					attrs.Timestamp++
				}
				if scenario == "txpool enabled" {
					attrs.NoTxPool = false
				}
				return attrs, nil
			})
			if scenario == "valid" {
				emitter.ExpectOnce(mock.MatchedBy(func(ev derive.DerivedAttributesEvent) bool {
					return ev.Attributes.Parent == parent && ev.Attributes.DerivedFrom == frontier && ev.Attributes.Concluding
				}))
				var queuedCtx context.Context
				f.emitter = event.EmitterFunc(func(ctx context.Context, ev event.Event) {
					queuedCtx = ctx
					emitter.Emit(ctx, ev)
				})
				require.NoError(t, f.next(t.Context(), parent))
				require.NotNil(t, queuedCtx)
				require.NoError(t, queuedCtx.Err(), "queued events must outlive the RPC deadline scope")
				require.NotNil(t, f.build)
				require.Equal(t, public, f.build.public)
				require.NoError(t, f.next(t.Context(), parent), "do not duplicate in-flight attributes")
			} else {
				require.Error(t, f.next(t.Context(), parent))
				require.Nil(t, f.build)
			}
			emitter.AssertExpectations(t)
		})
	}
}

func TestFollowRecoveryWithoutExtension(t *testing.T) {
	for _, private := range []bool{false, true} {
		t.Run(fmt.Sprintf("private_recovery_previously_enabled_%t", private), func(t *testing.T) {
			genesis := eth.L2BlockRef{Hash: common.Hash{1}}
			oldSafe := eth.L2BlockRef{Hash: common.Hash{3}, Number: 3}
			newSafe := eth.L2BlockRef{Hash: common.Hash{4}, Number: 4}
			unsafe := eth.L2BlockRef{Hash: common.Hash{5}, Number: 5}
			el := &testutils.MockEngine{}
			emitter := event.EmitterFunc(func(context.Context, event.Event) {})
			ec := engine.NewEngineController(t.Context(), el, testlog.Logger(t, 0), metrics.NoopMetrics,
				&rollup.Config{}, &syncconfig.Config{L2FollowSourceEndpoint: "http://localhost"}, &testutils.MockL1Source{}, emitter, nil)
			ec.SetUnsafeHead(unsafe)
			ec.SetLocalSafeHead(oldSafe)
			//nolint:staticcheck // Follow mode stores externally supplied cross-safe in this field.
			ec.SetDeprecatedSafeHead(oldSafe)
			ec.SetFinalizedHead(genesis)
			paused := false
			f := &followRecovery{enabled: private, engine: ec,
				pause: func(value bool) { paused = value },
				source: recoverySourceFunc(func(context.Context, uint64, eth.BlockID) (eth.L2BlockRef, error) {
					t.Fatal("a source without a recovery plan must not receive recoveryBlock calls")
					return eth.L2BlockRef{}, nil
				}),
			}
			if !private {
				el.ExpectL2BlockRefByNumber(newSafe.Number, newSafe, nil)
				el.ExpectForkchoiceUpdate(&eth.ForkchoiceState{
					HeadBlockHash: unsafe.Hash, SafeBlockHash: newSafe.Hash, FinalizedBlockHash: genesis.Hash,
				}, nil, &eth.ForkchoiceUpdatedResult{PayloadStatus: eth.PayloadStatusV1{Status: eth.ExecutionValid}}, nil)
			}
			err := f.update(t.Context(), &sources.FollowStatus{
				LocalSafeL2: newSafe, SafeL2: newSafe, FinalizedL2: genesis,
			})
			if private {
				require.ErrorContains(t, err, "omitted its snapshot")
				require.True(t, paused, "loss of a private plan must pause sequencing")
				require.Equal(t, oldSafe, ec.LocalSafeHead())
			} else {
				require.NoError(t, err)
				require.False(t, paused, "ordinary public following must not activate private recovery")
				require.False(t, f.enabled)
				require.Equal(t, newSafe, ec.LocalSafeHead())
				require.Equal(t, newSafe, ec.SafeL2Head())
			}
			require.Equal(t, unsafe, ec.UnsafeL2Head())
			require.Equal(t, genesis, ec.FinalizedHead())
			el.AssertExpectations(t)
		})
	}
}

type prefixL2 struct {
	L2Chain
	refs map[common.Hash]eth.L2BlockRef
}

func (l *prefixL2) L2BlockRefByHash(_ context.Context, hash common.Hash) (eth.L2BlockRef, error) {
	ref, ok := l.refs[hash]
	if !ok {
		return eth.L2BlockRef{}, fmt.Errorf("missing private header %s", hash)
	}
	return ref, nil
}

func TestFollowRecoveryAuthenticatesPrefixByPrivateAncestry(t *testing.T) {
	for _, scenario := range []string{"valid", "zero offset", "checkpoint boundary", "missing terminal", "missing ancestor", "wrong parent hash", "wrong parent number", "broken ancestry", "wrong public schedule", "conflicting checkpoint", "below finality", "negative offset"} {
		t.Run(scenario, func(t *testing.T) {
			l2 := &prefixL2{refs: make(map[common.Hash]eth.L2BlockRef)}
			chain := make([]eth.L2BlockRef, 9)
			for n := range chain {
				chain[n] = eth.L2BlockRef{Hash: common.Hash{byte(n + 1)}, Number: uint64(n), Time: uint64(100 + 2*n), SequenceNumber: uint64(n)}
				if n > 0 {
					chain[n].ParentHash = chain[n-1].Hash
				}
				l2.refs[chain[n].Hash] = chain[n]
			}
			base := chain[0]
			prefix := &sources.FollowRecoveryPrefix{Parent: chain[7].ID(), Last: chain[3]}
			want := chain[3]
			prefix.Last.Hash = common.Hash{0xff} // Public hash is deliberately different.
			switch scenario {
			case "zero offset":
				prefix.Last, want = chain[7], chain[7]
			case "checkpoint boundary":
				prefix.Last, want = base, base
			case "missing terminal":
				delete(l2.refs, chain[8].Hash)
			case "missing ancestor":
				delete(l2.refs, chain[5].Hash)
			case "wrong parent hash":
				prefix.Parent.Hash = common.Hash{0xff}
			case "wrong parent number":
				prefix.Parent.Number++
			case "broken ancestry":
				broken := chain[5]
				broken.Number--
				l2.refs[broken.Hash] = broken
			case "negative offset":
				prefix.Last = chain[8]
			case "wrong public schedule":
				prefix.Last.Time++
			case "conflicting checkpoint":
				base.Hash = common.Hash{0xee}
			case "below finality":
				base = chain[4]
			}
			f := &followRecovery{l2: l2}
			anchor, err := f.prefixAnchor(t.Context(), base, prefix)
			if scenario == "valid" || scenario == "zero offset" || scenario == "checkpoint boundary" || scenario == "missing terminal" {
				require.NoError(t, err)
				require.Equal(t, want, anchor)
			} else {
				require.Error(t, err)
			}
		})
	}
}

type recoveryBranchL2 struct {
	prefixL2
	canonical map[uint64]eth.L2BlockRef
}

func (l *recoveryBranchL2) L2BlockRefByNumber(_ context.Context, n uint64) (eth.L2BlockRef, error) {
	ref, ok := l.canonical[n]
	if !ok {
		return eth.L2BlockRef{}, fmt.Errorf("missing canonical block %d", n)
	}
	return ref, nil
}

func TestFollowRecoveryRestoresKnownBranchAndKeepsEventContext(t *testing.T) {
	for _, conflict := range []bool{false, true} {
		t.Run(fmt.Sprintf("conflicting_finality_%t", conflict), func(t *testing.T) {
			genesis := eth.L2BlockRef{Hash: common.Hash{1}, Time: 100}
			anchor := eth.L2BlockRef{Hash: common.Hash{2}, Number: 1, ParentHash: genesis.Hash, Time: 102}
			fork := anchor
			fork.Hash = common.Hash{3}
			l2 := &recoveryBranchL2{prefixL2: prefixL2{refs: map[common.Hash]eth.L2BlockRef{genesis.Hash: genesis, anchor.Hash: anchor}},
				canonical: map[uint64]eth.L2BlockRef{0: genesis, 1: fork}}
			el := &testutils.MockEngine{}
			var queued []context.Context
			em := event.EmitterFunc(func(ctx context.Context, _ event.Event) { queued = append(queued, ctx) })
			ec := engine.NewEngineController(t.Context(), el, testlog.Logger(t, 0), metrics.NoopMetrics,
				&rollup.Config{}, &syncconfig.Config{L2FollowSourceEndpoint: "http://localhost"}, &testutils.MockL1Source{}, em, nil)
			ec.SetUnsafeHead(fork)
			ec.SetLocalSafeHead(genesis)
			//nolint:staticcheck // Follow mode stores externally supplied cross-safe in this field.
			ec.SetDeprecatedSafeHead(genesis)
			retained := genesis
			if conflict {
				retained.Hash = common.Hash{0xff}
			}
			ec.SetFinalizedHead(retained)
			paused := false
			f := &followRecovery{l2: l2, engine: ec, emitter: em, pause: func(value bool) { paused = value }}
			status := &sources.FollowStatus{LocalSafeL2: anchor, SafeL2: anchor, FinalizedL2: genesis,
				CurrentL1: eth.L1BlockRef{Hash: common.Hash{5}, Number: 10},
				Recovery: &sources.FollowRecoveryStatus{Anchor: anchor, Target: eth.L2BlockRef{Hash: common.Hash{6}, Number: 2},
					Safe: eth.L2BlockRef{Hash: common.Hash{7}, Number: 1}, Finalized: genesis}}
			if !conflict {
				el.On("ForkchoiceUpdate", mock.Anything, mock.Anything).
					Return(&eth.ForkchoiceUpdatedResult{PayloadStatus: eth.PayloadStatusV1{Status: eth.ExecutionValid}}, nil).
					Run(func(mock.Arguments) { l2.canonical[1] = anchor })
				el.ExpectL2BlockRefByNumber(anchor.Number, anchor, nil)
			}
			err := f.update(t.Context(), status)
			require.True(t, paused, "pause until canonical replacements are reconciled")
			if conflict {
				require.ErrorContains(t, err, "finalized ancestry")
				require.Equal(t, fork, ec.UnsafeL2Head())
				require.Empty(t, queued)
			} else {
				require.NoError(t, err)
				require.Equal(t, anchor, ec.UnsafeL2Head())
				require.Equal(t, anchor, ec.PendingSafeL2Head())
				require.NotEmpty(t, queued)
				for _, ctx := range queued {
					require.NoError(t, ctx.Err(), "RPC deadlines must not cancel queued engine events")
				}
			}
			el.AssertExpectations(t)
		})
	}
}

func TestFollowRecoveryWaitsForWholeReplacementInterval(t *testing.T) {
	plan := &sources.FollowRecoveryStatus{
		Target: eth.L2BlockRef{Number: 6},
		Prefix: &sources.FollowRecoveryPrefix{Parent: eth.BlockID{Number: 7}, Last: eth.L2BlockRef{Number: 3}},
	}
	f := &followRecovery{status: &sources.FollowStatus{Recovery: plan}}
	number := uint64(6)
	require.False(t, f.canResume(number), "a surviving claim still reserves block 8 while fallback has only reached 6")
	plan.Target.Number, number = 7, 7
	require.False(t, f.canResume(number), "the committed parent is still before the final replacement position")
	plan.Target.Number, number = 8, 8
	require.True(t, f.canResume(number), "resume after executing the whole reserved range")
	plan.Target.Number = 9
	require.False(t, f.canResume(number), "also execute any later canonical fallback")
	number = 9
	require.True(t, f.canResume(number))
}

func TestFollowRecoveryRejectsLegacyPrefixBeforeReset(t *testing.T) {
	// An old source's terminal/terminal_parent fields do not populate Parent.
	// Reject this even when private finality could otherwise skip prefix lookup.
	status := &sources.FollowStatus{
		CurrentL1: eth.L1BlockRef{Number: 10},
		Recovery: &sources.FollowRecoveryStatus{
			Target: eth.L2BlockRef{Number: 8},
			Prefix: &sources.FollowRecoveryPrefix{Last: eth.L2BlockRef{Number: 3}},
		},
	}
	paused := false
	f := &followRecovery{pause: func(value bool) { paused = value }}
	require.ErrorContains(t, f.update(t.Context(), status), "invalid surviving prefix bounds")
	require.True(t, paused)
}

func TestCanonicalRecoveryCheckpointDoesNotWalkToGenesis(t *testing.T) {
	genesis := eth.L2BlockRef{Hash: common.Hash{1}}
	anchor := eth.L2BlockRef{Hash: common.Hash{2}, ParentHash: common.Hash{3}, Number: 1_000_000}
	unsafe := eth.L2BlockRef{Hash: common.Hash{4}, Number: 1_000_300}
	l2 := &recoveryBranchL2{
		prefixL2:  prefixL2{refs: map[common.Hash]eth.L2BlockRef{anchor.Hash: anchor}},
		canonical: map[uint64]eth.L2BlockRef{0: genesis, anchor.Number: anchor},
	}
	el := &testutils.MockEngine{}
	em := event.EmitterFunc(func(context.Context, event.Event) {})
	ec := engine.NewEngineController(t.Context(), el, testlog.Logger(t, 0), metrics.NoopMetrics,
		&rollup.Config{}, &syncconfig.Config{L2FollowSourceEndpoint: "http://localhost"}, &testutils.MockL1Source{}, em, nil)
	ec.SetUnsafeHead(unsafe)
	ec.SetLocalSafeHead(genesis)
	ec.SetPendingSafeL2Head(genesis)
	ec.SetFinalizedHead(genesis)
	//nolint:staticcheck // Follow mode stores external cross-safe here.
	ec.SetDeprecatedSafeHead(genesis)
	el.ExpectL2BlockRefByNumber(anchor.Number, anchor, nil)
	el.On("ForkchoiceUpdate", mock.Anything, mock.Anything).Return(&eth.ForkchoiceUpdatedResult{PayloadStatus: eth.PayloadStatusV1{Status: eth.ExecutionValid}}, nil)
	f := &followRecovery{l2: l2, engine: ec, pause: func(bool) {}}
	status := &sources.FollowStatus{LocalSafeL2: anchor, SafeL2: anchor, FinalizedL2: genesis,
		Recovery: &sources.FollowRecoveryStatus{Anchor: anchor, Target: anchor}}
	require.NoError(t, f.adopt(t.Context(), t.Context(), status))
	require.Equal(t, anchor, ec.LocalSafeHead())
	require.Equal(t, anchor, ec.PendingSafeL2Head())
	require.Equal(t, unsafe, ec.UnsafeL2Head(), "a completed checkpoint must preserve the sequenced suffix")
	el.AssertExpectations(t)
}
