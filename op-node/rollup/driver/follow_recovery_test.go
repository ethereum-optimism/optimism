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
			f := &followRecovery{mapped: parent, emitter: emitter, status: &sources.FollowStatus{
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
				require.Equal(t, public, f.inflight)
				require.NoError(t, f.next(t.Context(), parent), "do not duplicate in-flight attributes")
			} else {
				require.Error(t, f.next(t.Context(), parent))
				require.Zero(t, f.inflight)
			}
			emitter.AssertExpectations(t)
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
	for _, scenario := range []string{"valid", "missing ancestor", "wrong terminal parent", "wrong public schedule", "conflicting checkpoint", "below finality"} {
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
			prefix := &sources.FollowRecoveryPrefix{Terminal: chain[8].ID(), TerminalParent: chain[7].Hash, Last: chain[3]}
			prefix.Last.Hash = common.Hash{0xff} // Public hash is deliberately different.
			switch scenario {
			case "missing ancestor":
				delete(l2.refs, chain[5].Hash)
			case "wrong terminal parent":
				prefix.TerminalParent = common.Hash{0xff}
			case "wrong public schedule":
				prefix.Last.Time++
			case "conflicting checkpoint":
				base.Hash = common.Hash{0xee}
			case "below finality":
				base = chain[4]
			}
			f := &followRecovery{l2: l2}
			anchor, err := f.prefixAnchor(t.Context(), base, prefix)
			if scenario == "valid" {
				require.NoError(t, err)
				require.Equal(t, chain[3], anchor)
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
