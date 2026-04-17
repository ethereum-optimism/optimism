package engine

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-node/metrics"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/sync"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/ethereum/go-ethereum/common"
)

// TestMaybeSkipELSyncIfEngineAlreadySynced verifies the startup guard for
// #18468: when op-node starts in --syncmode=execution-layer against an
// already-synced engine, syncStatus must transition to syncStatusFinishedEL
// and a ResetEngineRequestEvent must be emitted so heads are initialized
// from the engine via FindL2Heads. Otherwise op-node stalls in
// syncStatusWillStartEL waiting for a fresh unsafe payload that never
// arrives.
func TestMaybeSkipELSyncIfEngineAlreadySynced(t *testing.T) {
	genesisHash := common.HexToHash("0xdeadbeef")
	genesisRef := eth.L2BlockRef{Hash: genesisHash, Number: 0}
	finalizedRef := eth.L2BlockRef{Hash: common.HexToHash("0xcafe"), Number: 1000}

	cfg := &rollup.Config{
		Genesis: rollup.Genesis{
			L2: eth.BlockID{Hash: genesisHash, Number: 0},
		},
	}

	t.Run("engine synced: transitions to FinishedEL and emits reset", func(t *testing.T) {
		mockEngine := &testutils.MockEngine{}
		mockEngine.ExpectL2BlockRefByLabel(eth.Finalized, finalizedRef, nil)

		emitter := &testutils.MockEmitter{}
		emitter.ExpectOnce(ResetEngineRequestEvent{})

		ec := NewEngineController(context.Background(), mockEngine, testlog.Logger(t, 0),
			metrics.NoopMetrics, cfg, &sync.Config{SyncMode: sync.ELSync}, false,
			&testutils.MockL1Source{}, emitter, nil)
		require.Equal(t, syncStatusWillStartEL, ec.syncStatus)

		require.NoError(t, ec.MaybeSkipELSyncIfEngineAlreadySynced(context.Background()))

		require.Equal(t, syncStatusFinishedEL, ec.syncStatus)
		mockEngine.AssertExpectations(t)
		emitter.AssertExpectations(t)
	})

	t.Run("engine at genesis finalized: stays in WillStartEL", func(t *testing.T) {
		mockEngine := &testutils.MockEngine{}
		mockEngine.ExpectL2BlockRefByLabel(eth.Finalized, genesisRef, nil)

		emitter := &testutils.MockEmitter{}
		// No Emit expectation — any Emit call would fail AssertExpectations.

		ec := NewEngineController(context.Background(), mockEngine, testlog.Logger(t, 0),
			metrics.NoopMetrics, cfg, &sync.Config{SyncMode: sync.ELSync}, false,
			&testutils.MockL1Source{}, emitter, nil)

		require.NoError(t, ec.MaybeSkipELSyncIfEngineAlreadySynced(context.Background()))

		require.Equal(t, syncStatusWillStartEL, ec.syncStatus)
		mockEngine.AssertExpectations(t)
		emitter.AssertExpectations(t)
	})

	t.Run("SupportsPostFinalizationELSync: stays in WillStartEL", func(t *testing.T) {
		mockEngine := &testutils.MockEngine{}
		mockEngine.ExpectL2BlockRefByLabel(eth.Finalized, finalizedRef, nil)

		emitter := &testutils.MockEmitter{}

		ec := NewEngineController(context.Background(), mockEngine, testlog.Logger(t, 0),
			metrics.NoopMetrics, cfg,
			&sync.Config{SyncMode: sync.ELSync, SupportsPostFinalizationELSync: true}, false,
			&testutils.MockL1Source{}, emitter, nil)

		require.NoError(t, ec.MaybeSkipELSyncIfEngineAlreadySynced(context.Background()))

		require.Equal(t, syncStatusWillStartEL, ec.syncStatus)
		mockEngine.AssertExpectations(t)
		emitter.AssertExpectations(t)
	})

	t.Run("not in WillStartEL: no-op, engine not queried", func(t *testing.T) {
		mockEngine := &testutils.MockEngine{}
		// No ExpectL2BlockRefByLabel — engine must not be queried.

		emitter := &testutils.MockEmitter{}

		ec := NewEngineController(context.Background(), mockEngine, testlog.Logger(t, 0),
			metrics.NoopMetrics, cfg, &sync.Config{SyncMode: sync.CLSync}, false,
			&testutils.MockL1Source{}, emitter, nil)
		require.Equal(t, syncStatusCL, ec.syncStatus)

		require.NoError(t, ec.MaybeSkipELSyncIfEngineAlreadySynced(context.Background()))

		require.Equal(t, syncStatusCL, ec.syncStatus)
		mockEngine.AssertExpectations(t)
		emitter.AssertExpectations(t)
	})
}
