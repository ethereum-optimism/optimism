package sysgo

import (
	"context"
	"errors"
	"syscall"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/devtest"
	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/shim"
	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/stack"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/retry"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/status"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

// TestControlPlane tests start/stop functionality provided control plane
func TestControlPlane(gt *testing.T) {
	var ids DefaultInteropSystemIDs
	opt := DefaultInteropSystem(&ids)

	logger := testlog.Logger(gt, log.LevelInfo)

	p := devtest.NewP(
		context.Background(),
		logger,
		func() {
			gt.Helper()
			gt.FailNow()
		},
	)
	gt.Cleanup(p.Close)

	orch := NewOrchestrator(p)
	opt(orch)

	control := orch.ControlPlane()

	gt.Run("test-SupervisorRestart", func(gt *testing.T) {
		gt.Parallel()

		t := devtest.SerialT(gt)
		system := shim.NewSystem(t)
		orch.Hydrate(system)

		testSupervisorRestart(ids, system, control)
	})

	gt.Run("test-L2CLRestart", func(gt *testing.T) {
		gt.Parallel()

		t := devtest.SerialT(gt)
		system := shim.NewSystem(t)
		orch.Hydrate(system)

		testL2CLRestart(ids, system, control)
	})
}

func testSupervisorRestart(ids DefaultInteropSystemIDs, system stack.System, control stack.ControlPlane) {
	t := system.T()
	logger := t.Logger()
	supervisor := system.Supervisor(ids.Supervisor)

	// progress supervisor
	for range 3 {
		time.Sleep(time.Second * 2)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		status, err := supervisor.QueryAPI().SyncStatus(ctx)
		require.NoError(t, err)
		cancel()
		logger.Info("supervisor L1 view", "tip", status.MinSyncedL1)
	}

	// stop supervisor
	control.SupervisorState(ids.Supervisor, stack.Stop)

	// supervisor API will not work since L2CL stopped
	{
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		_, err := retry.Do[eth.SupervisorSyncStatus](ctx, 10, retry.Fixed(time.Millisecond*500), func() (eth.SupervisorSyncStatus, error) {
			return supervisor.QueryAPI().SyncStatus(ctx)
		})
		cancel()
		require.True(t, errors.Is(err, syscall.ECONNREFUSED))
	}

	// restart supervisor
	control.SupervisorState(ids.Supervisor, stack.Start)

	// check supervisor API is back
	{
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		_, err := retry.Do[eth.SupervisorSyncStatus](ctx, 3, retry.Fixed(time.Millisecond*500), func() (eth.SupervisorSyncStatus, error) {
			return supervisor.QueryAPI().SyncStatus(ctx)
		})
		if err != nil {
			// API is still back, although supervisor status tracker not ready
			require.Equal(t, errors.Unwrap(err).Error(), status.ErrStatusTrackerNotReady.Error())
		}
		cancel()
	}
}

func testL2CLRestart(ids DefaultInteropSystemIDs, system stack.System, control stack.ControlPlane) {
	t := system.T()
	logger := t.Logger()
	seqA := system.L2Network(ids.L2A).L2CLNode(ids.L2ACL)

	// progress chain
	for range 3 {
		time.Sleep(time.Second * 2)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		status, err := seqA.RollupAPI().SyncStatus(ctx)
		require.NoError(t, err)
		cancel()
		logger.Info("chain A", "tip", status.UnsafeL2)
	}

	// stop L2CL
	control.L2CLNodeState(ids.L2ACL, stack.Stop)

	// L2CL API will not work since L2CL stopped
	{
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		_, err := retry.Do[*eth.SyncStatus](ctx, 10, retry.Fixed(time.Millisecond*500), func() (*eth.SyncStatus, error) {
			return seqA.RollupAPI().SyncStatus(ctx)
		})
		cancel()
		require.True(t, errors.Is(err, syscall.ECONNREFUSED))
	}

	// restart L2CL
	control.L2CLNodeState(ids.L2ACL, stack.Start)

	// check L2CL API is back
	{
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		_, err := retry.Do[*eth.SyncStatus](ctx, 3, retry.Fixed(time.Millisecond*500), func() (*eth.SyncStatus, error) {
			return seqA.RollupAPI().SyncStatus(ctx)
		})
		require.NoError(t, err)
		cancel()
	}
}
