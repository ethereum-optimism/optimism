package driver

import (
	"context"
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

func TestRecoveryJournalRetainsPrefixAcrossRestart(t *testing.T) {
	l2 := &prefixL2{refs: make(map[common.Hash]eth.L2BlockRef)}
	chain := make([]eth.L2BlockRef, 8)
	for n := range chain {
		chain[n] = eth.L2BlockRef{Hash: common.Hash{byte(n + 1)}, Number: uint64(n), Time: uint64(100 + 2*n), SequenceNumber: uint64(n)}
		if n > 0 {
			chain[n].ParentHash = chain[n-1].Hash
		}
		l2.refs[chain[n].Hash] = chain[n]
	}
	path := filepath.Join(t.TempDir(), "private", "recovery.db")
	f := &followRecovery{l2: l2, journal: &recoveryJournal{path: path, genesis: chain[0].Hash}}
	prefix := &sources.FollowRecoveryPrefix{Parent: chain[7].ID(), Last: chain[4]}
	prefix.Last.Hash = common.Hash{0xff}
	anchor, err := f.prefixAnchor(t.Context(), chain[0], prefix)
	require.NoError(t, err)
	require.Equal(t, chain[4], anchor)
	require.NoError(t, f.journal.commit(0))
	// All original private headers disappear, just as noncanonical EL headers can.
	clear(l2.refs)
	f = &followRecovery{l2: l2, journal: &recoveryJournal{path: path, genesis: chain[0].Hash}}
	got, err := f.prefixAnchor(t.Context(), chain[0], prefix)
	require.NoError(t, err)
	require.Equal(t, anchor, got)
	// A second invalidation can shrink the same claim's surviving prefix.
	prefix.Last = chain[2]
	got, err = f.prefixAnchor(t.Context(), chain[0], prefix)
	require.NoError(t, err)
	require.Equal(t, chain[2], got)
	prefix.Last.Time++
	_, err = f.prefixAnchor(t.Context(), chain[0], prefix)
	require.ErrorContains(t, err, "surviving projection inputs")
	require.NoError(t, f.journal.commit(3))
	reopened := &recoveryJournal{path: path, genesis: chain[0].Hash}
	require.NoError(t, reopened.load())
	require.Len(t, reopened.headers, 5)
	for _, ref := range reopened.headers {
		require.GreaterOrEqual(t, ref.Number, uint64(3))
	}
	wrongChain := &recoveryJournal{path: path, genesis: common.Hash{9}}
	require.ErrorContains(t, wrongChain.load(), "another private genesis")
}

func TestRecoveryJournalRequiresWritableDurablePath(t *testing.T) {
	var missing *recoveryJournal
	require.ErrorContains(t, missing.commit(0), "recovery-path")
	path := filepath.Join(t.TempDir(), "not-a-directory")
	require.NoError(t, os.WriteFile(path, []byte("occupied"), 0600))
	journal := &recoveryJournal{path: filepath.Join(path, "recovery.db"), headers: map[common.Hash]eth.L2BlockRef{}}
	require.Error(t, journal.commit(0))
	// Corruption cannot silently replace durable ancestry with an empty cache.
	journal = &recoveryJournal{path: path}
	require.Error(t, journal.load())
	require.Nil(t, journal.headers)
}

func TestRecoveryJournalFailurePreventsRewind(t *testing.T) {
	base := eth.L2BlockRef{Hash: common.Hash{1}, Time: 100}
	prefix := eth.L2BlockRef{Hash: common.Hash{2}, Number: 1, ParentHash: base.Hash, Time: 102}
	tip := eth.L2BlockRef{Hash: common.Hash{3}, Number: 2, ParentHash: prefix.Hash, Time: 104}
	l2 := &recoveryBranchL2{prefixL2: prefixL2{refs: map[common.Hash]eth.L2BlockRef{base.Hash: base, prefix.Hash: prefix, tip.Hash: tip}}, canonical: map[uint64]eth.L2BlockRef{0: base, 1: prefix, 2: tip}}
	el := &testutils.MockEngine{}
	em := event.EmitterFunc(func(context.Context, event.Event) {})
	ec := engine.NewEngineController(t.Context(), el, testlog.Logger(t, 0), metrics.NoopMetrics, &rollup.Config{}, &syncconfig.Config{L2FollowSourceEndpoint: "http://localhost"}, &testutils.MockL1Source{}, em, nil)
	ec.SetUnsafeHead(tip)
	ec.SetLocalSafeHead(tip)
	ec.SetFinalizedHead(base)
	//nolint:staticcheck // Follow mode stores externally supplied cross-safe here.
	ec.SetDeprecatedSafeHead(base)
	paused := false
	f := &followRecovery{l2: l2, engine: ec, pause: func(v bool) { paused = v }, journal: &recoveryJournal{}}
	status := &sources.FollowStatus{LocalSafeL2: base, SafeL2: base, FinalizedL2: base, CurrentL1: eth.L1BlockRef{Hash: common.Hash{9}, Number: 10},
		Recovery: &sources.FollowRecoveryStatus{Anchor: base, Target: tip, Safe: base, Finalized: base, Prefix: &sources.FollowRecoveryPrefix{Parent: tip.ID(), Last: prefix}}}
	require.ErrorContains(t, f.update(t.Context(), status), "recovery-path")
	require.True(t, paused)
	require.Equal(t, tip, ec.UnsafeL2Head())
	require.Equal(t, tip, ec.LocalSafeHead())
	el.AssertNotCalled(t, "ForkchoiceUpdate", mock.Anything, mock.Anything)
}
