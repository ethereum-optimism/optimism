package sysgo

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/shim"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-devstack/stack/match"
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
	onFail, onSkipNow := exiters(gt)

	p := devtest.NewP(context.Background(), logger, onFail, onSkipNow)
	gt.Cleanup(p.Close)

	orch := NewOrchestrator(p, stack.Combine[*Orchestrator]())
	stack.ApplyOptionLifecycle(opt, orch)

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
		require.Error(t, err)
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

	// L2CL API will still kind of work, it is not functioning,
	// but since L2CL is behind a proxy, the proxy is still online, and may create a different error.
	// The dial will be accepted, and the connection then closed, once the connection behind the proxy fails.
	{
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		_, err := retry.Do[*eth.SyncStatus](ctx, 10, retry.Fixed(time.Millisecond*500), func() (*eth.SyncStatus, error) {
			return seqA.RollupAPI().SyncStatus(ctx)
		})
		cancel()
		require.Error(t, err, "should not be able to get sync-status when node behind proxy is offline")
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

// TestControlPlaneFakePoS tests the start/stop functionality provided by the control plane for the fakePoS module
func TestControlPlaneFakePoS(gt *testing.T) {
	var ids DefaultInteropSystemIDs
	opt := DefaultInteropSystem(&ids)

	logger := testlog.Logger(gt, log.LevelInfo)
	onFail, onSkipNow := exiters(gt)
	p := devtest.NewP(context.Background(), logger, onFail, onSkipNow)
	gt.Cleanup(p.Close)

	orch := NewOrchestrator(p, stack.Combine[*Orchestrator]())
	stack.ApplyOptionLifecycle(opt, orch)

	control := orch.ControlPlane()

	t := devtest.SerialT(gt)
	system := shim.NewSystem(t)
	orch.Hydrate(system)

	ctx := t.Ctx()

	el := system.L1Network(ids.L1).L1ELNode(match.FirstL1EL)

	// progress chain
	blockTime := time.Second * 6
	for range 2 {
		time.Sleep(blockTime)

		head, err := el.EthClient().InfoByLabel(ctx, "latest")
		require.NoError(t, err)
		logger.Info("L1 chain", "number", head.NumberU64(), "hash", head.Hash())
	}

	logger.Info("Stopping fakePoS service")
	control.FakePoSState(ids.L1CL, stack.Stop)

	head, err := el.EthClient().InfoByLabel(ctx, "latest")
	require.NoError(t, err)

	// L1 chain won't progress since fakePoS is stopped
	// Wait and check that L1 chain won't progress
	for range 2 {
		time.Sleep(blockTime)

		other, err := el.EthClient().InfoByLabel(ctx, "latest")
		require.NoError(t, err)
		logger.Info("L1 chain", "number", other.NumberU64(), "hash", other.Hash(), "previous", head.Hash())

		require.Equal(t, other.Hash(), head.Hash())
	}

	// Restart fakePoS
	logger.Info("Starting fakePoS service")
	control.FakePoSState(ids.L1CL, stack.Start)

	// L1 chain should progress again eventually
	require.Eventually(t, func() bool {
		other, err := el.EthClient().InfoByLabel(ctx, "latest")
		require.NoError(t, err)
		logger.Info("L1 chain", "number", other.NumberU64(), "hash", other.Hash(), "previous", head.Hash())

		return other.Hash() != head.Hash() && other.NumberU64() > head.NumberU64()
	}, time.Second*20, time.Second*2)
}
