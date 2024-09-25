package devnet

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-e2e/system/bridge"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

func TestDevnet(t *testing.T) {
	lgr := testlog.Logger(t, slog.LevelDebug)
	ctx, done := context.WithTimeout(context.Background(), time.Minute)
	defer done()

	sys, err := NewSystem(ctx, lgr)
	require.NoError(t, err)

	t.Run("SyncFinalized", func(t *testing.T) {
		// SyncFinalized can run in parallel to Withdrawals test, because propopser
		// already posts unfinalized output roots in devnet mode.
		t.Parallel()
		testSyncFinalized(t, sys)
	})
	t.Run("Withdrawal", func(t *testing.T) {
		t.Parallel()
		bridge.RunWithdrawalsTest(t, sys)
	})
}

func testSyncFinalized(t *testing.T, sys *System) {
	const timeout = 4 * time.Minute
	ctx, done := context.WithTimeout(context.Background(), timeout)
	defer done()

	require.EventuallyWithT(t, func(tc *assert.CollectT) {
		ss, err := sys.Rollup.SyncStatus(ctx)
		assert.NoError(tc, err)
		if err != nil {
			t.Log(err)
			return
		}
		t.Logf("SyncStatus: %+v", ss)
		assert.NotZero(tc, ss.FinalizedL2.Number)
	}, timeout, 2*time.Second)
}
