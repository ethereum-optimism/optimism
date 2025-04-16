package sysgo

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/devtest"
	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/shim"
	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/stack"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

// TestL2CLResync checks that unsafe head advances after restarting L2CL
func TestL2CLResync(gt *testing.T) {
	var ids DefaultInteropSystemIDs
	opt := DefaultInteropSystem(&ids)

	logger := testlog.Logger(gt, log.LevelInfo)

	p := devtest.NewP(logger, func() {
		gt.Helper()
		gt.FailNow()
	})
	gt.Cleanup(p.Close)

	orch := NewOrchestrator(p)
	opt(orch)

	t := devtest.SerialT(gt)
	system := shim.NewSystem(t)
	orch.Hydrate(system)

	control := orch.ControlPlane()

	{
		logger := system.T().Logger()

		elA := system.L2Network(ids.L2A).L2ELNode(ids.L2AEL)
		elB := system.L2Network(ids.L2B).L2ELNode(ids.L2BEL)

		query := func() (eth.BlockRef, eth.BlockRef) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
			blockA, err := elA.EthClient().BlockRefByLabel(ctx, "latest")
			require.NoError(t, err)
			blockB, err := elB.EthClient().BlockRefByLabel(ctx, "latest")
			require.NoError(t, err)
			cancel()
			logger.Info("chain A", "blockNum", blockA.Number, "tip", blockA)
			logger.Info("chain B", "blockNum", blockB.Number, "tip", blockB)
			return blockA, blockB
		}

		logger.Info("wait until passing genesis")
		for range 5 {
			query()
			time.Sleep(time.Millisecond * 2500)
		}

		logger.Info("check unsafe chains are advancing")
		prevBlockA, prevBlockB := query()
		time.Sleep(time.Millisecond * 2500)
		for range 5 {
			blockA, blockB := query()
			require.Greater(t, blockA.Number, prevBlockA.Number)
			require.Greater(t, blockB.Number, prevBlockB.Number)
			prevBlockA, prevBlockB = blockA, blockB
			time.Sleep(time.Millisecond * 2500)
		}

		logger.Info("stop CL nodes")
		control.L2CLNodeState(ids.L2ACL, stack.Stop)
		control.L2CLNodeState(ids.L2BCL, stack.Stop)

		logger.Info("make sure ELs does not advance")
		pausedBlockA, pausedBlockB := query()
		for range 5 {
			blockA, blockB := query()
			require.Equal(t, pausedBlockA.Hash, blockA.Hash)
			require.Equal(t, pausedBlockB.Hash, blockB.Hash)
			time.Sleep(time.Millisecond * 2500)
		}

		logger.Info("restart CL nodes")
		control.L2CLNodeState(ids.L2ACL, stack.Start)
		control.L2CLNodeState(ids.L2BCL, stack.Start)

		// supervisor will attempt to reconnect with L2CLs at this point because L2CL ws endpoint is recovered
		logger.Info("wait until L2CLs heat up again")
		for range 5 {
			query()
			time.Sleep(time.Millisecond * 2500)
		}

		logger.Info("check unsafe chains are advancing again")
		prevBlockA, prevBlockB = pausedBlockA, pausedBlockB
		for range 5 {
			blockA, blockB := query()
			require.Greater(t, blockA.Number, prevBlockA.Number)
			require.Greater(t, blockB.Number, prevBlockB.Number)
			prevBlockA, prevBlockB = blockA, blockB
			time.Sleep(time.Millisecond * 2500)
		}
		// supervisor is successfully connected with managed L2CLs
	}
}
