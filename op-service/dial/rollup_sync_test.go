package dial

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/stretchr/testify/require"
)

func TestWaitRollupSync(t *testing.T) {
	bctx := context.Background()
	hasInfoLevel := testlog.NewLevelFilter(slog.LevelInfo)
	const target = 42

	t.Run("sync-error", func(t *testing.T) {
		lgr, logs := testlog.CaptureLogger(t, slog.LevelInfo)
		rollup := new(testutils.MockRollupClient)
		syncErr := errors.New("test sync error")
		rollup.ExpectSyncStatus(nil, syncErr)

		err := WaitRollupSync(bctx, lgr, rollup, 0, 0)
		require.ErrorIs(t, err, syncErr)
		require.Nil(t, logs.FindLog(hasInfoLevel), "expected no logs")
		rollup.AssertExpectations(t)
	})

	t.Run("at-target", func(t *testing.T) {
		lgr, logs := testlog.CaptureLogger(t, slog.LevelDebug)
		rollup := new(testutils.MockRollupClient)
		rollup.ExpectSyncStatus(&eth.SyncStatus{
			CurrentL1: eth.L1BlockRef{Number: target},
		}, nil)

		err := WaitRollupSync(bctx, lgr, rollup, target, 0)
		require.NoError(t, err)
		require.NotNil(t, logs.FindLog(hasInfoLevel,
			testlog.NewMessageContainsFilter("target reached")))
		rollup.AssertExpectations(t)
	})

	t.Run("beyond-target", func(t *testing.T) {
		lgr, logs := testlog.CaptureLogger(t, slog.LevelDebug)
		rollup := new(testutils.MockRollupClient)
		rollup.ExpectSyncStatus(&eth.SyncStatus{
			CurrentL1: eth.L1BlockRef{Number: target + 12},
		}, nil)

		err := WaitRollupSync(bctx, lgr, rollup, target, 0)
		require.NoError(t, err)
		require.NotNil(t, logs.FindLog(hasInfoLevel,
			testlog.NewMessageContainsFilter("target reached")))
		rollup.AssertExpectations(t)
	})

	t.Run("few-blocks-before-target", func(t *testing.T) {
		lgr, logs := testlog.CaptureLogger(t, slog.LevelDebug)
		rollup := new(testutils.MockRollupClient)
		const gap = 7
		for i := -gap; i <= 0; i++ {
			rollup.ExpectSyncStatus(&eth.SyncStatus{
				CurrentL1: eth.L1BlockRef{Number: uint64(target + i)},
			}, nil)
		}

		err := WaitRollupSync(bctx, lgr, rollup, target, 0)
		require.NoError(t, err)
		require.NotNil(t, logs.FindLog(hasInfoLevel,
			testlog.NewMessageContainsFilter("target reached")))
		require.Len(t, logs.FindLogs(hasInfoLevel,
			testlog.NewMessageContainsFilter("retrying")), gap)
		rollup.AssertExpectations(t)
	})

	t.Run("ctx-timeout", func(t *testing.T) {
		lgr, logs := testlog.CaptureLogger(t, slog.LevelDebug)
		rollup := new(testutils.MockRollupClient)
		rollup.ExpectSyncStatus(&eth.SyncStatus{
			CurrentL1: eth.L1BlockRef{Number: uint64(target - 1)},
		}, nil)

		ctx, cancel := context.WithCancel(bctx)
		// We can already cancel the context because the mock ignores the
		// cancelled context.
		cancel()
		// need real duration or the timer races with the cancelled context
		err := WaitRollupSync(ctx, lgr, rollup, target, time.Second)
		require.ErrorIs(t, err, context.Canceled)
		require.NotNil(t, logs.FindLogs(hasInfoLevel,
			testlog.NewMessageContainsFilter("retrying")))
		require.NotNil(t, logs.FindLog(
			testlog.NewLevelFilter(slog.LevelWarn),
			testlog.NewMessageContainsFilter("timed out")))
		rollup.AssertExpectations(t)
	})
}
