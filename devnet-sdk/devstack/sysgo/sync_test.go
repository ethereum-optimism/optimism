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

// TestL2CLResync checks that unsafe head advances after restarting L2CL.
// Resync is only possible when supervisor and L2CL reconnects.
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

	blockTime := system.L2Network(ids.L2A).RollupConfig().BlockTime
	require.Equal(t, blockTime, system.L2Network(ids.L2B).RollupConfig().BlockTime)

	waitTime := time.Duration(blockTime+1) * time.Second
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
		var prevBlockA, prevBlockB eth.BlockRef
		require.Eventually(t, func() bool {
			blockA, blockB := query()
			prevBlockA, prevBlockB = blockA, blockB
			return blockA.Number > 0 && blockB.Number > 0
		}, 16*time.Second, waitTime)

		time.Sleep(waitTime)
		logger.Info("check unsafe chains are advancing")
		require.Never(t, func() bool {
			blockA, blockB := query()
			advanced := prevBlockA.Number < blockA.Number && prevBlockB.Number < blockB.Number
			prevBlockA, prevBlockB = blockA, blockB
			return !advanced
		}, 10*time.Second, waitTime)

		logger.Info("stop L2CL nodes")
		control.L2CLNodeState(ids.L2ACL, stack.Stop)
		control.L2CLNodeState(ids.L2BCL, stack.Stop)

		logger.Info("make sure L2ELs does not advance")
		require.Eventually(t, func() bool {
			blockA, blockB := query()
			isStatic := prevBlockA.Hash == blockA.Hash && prevBlockB.Hash == blockB.Hash
			prevBlockA, prevBlockB = blockA, blockB
			return isStatic
		}, 10*time.Second, waitTime)

		logger.Info("restart L2CL nodes")
		control.L2CLNodeState(ids.L2ACL, stack.Start)
		control.L2CLNodeState(ids.L2BCL, stack.Start)

		// L2CL may advance a few blocks without supervisor connection, but eventually it will stop without the connection
		// we must check that unsafe head is advancing due to reconnection
		logger.Info("boot up L2CL nodes")
		require.Eventually(t, func() bool {
			blockA, blockB := query()
			advanced := prevBlockA.Number < blockA.Number && prevBlockB.Number < blockB.Number
			prevBlockA, prevBlockB = blockA, blockB
			return advanced
		}, 15*time.Second, waitTime)

		// supervisor will attempt to reconnect with L2CLs at this point because L2CL ws endpoint is recovered
		logger.Info("check unsafe chains are advancing again")
		require.Never(t, func() bool {
			blockA, blockB := query()
			advanced := prevBlockA.Number < blockA.Number && prevBlockB.Number < blockB.Number
			prevBlockA, prevBlockB = blockA, blockB
			return !advanced
		}, 15*time.Second, waitTime)

		// supervisor successfully connected with managed L2CLs
	}
}

// TestL2CLSyncP2P checks that unsafe head is propagated from sequencer to verifier.
// Tests started/restarted L2CL advances unsafe head via P2P connection.
func TestL2CLSyncP2P(gt *testing.T) {
	var ids DefaultRedundancyInteropSystemIDs
	opt := DefaultRedundancyInteropSystem(&ids)

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

	control := orch.controlPlane

	blockTime := system.L2Network(ids.L2A).RollupConfig().BlockTime

	waitTime := time.Duration(blockTime+1) * time.Second
	{
		logger := system.T().Logger()

		elA := system.L2Network(ids.L2A).L2ELNode(ids.L2AEL)
		elA2 := system.L2Network(ids.L2A).L2ELNode(ids.L2A2EL)

		queryLatest := func() (eth.BlockRef, eth.BlockRef) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
			blockA, err := elA.EthClient().BlockRefByLabel(ctx, "latest")
			require.NoError(t, err)
			blockA2, err := elA2.EthClient().BlockRefByLabel(ctx, "latest")
			require.NoError(t, err)
			cancel()
			logger.Info("chain A", "blockNum", blockA.Number, "tip", blockA)
			logger.Info("chain A2", "blockNum", blockA2.Number, "tip", blockA2)
			return blockA, blockA2
		}

		queryBlock := func(blockNum uint64) (eth.BlockRef, eth.BlockRef) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
			blockA, err := elA.EthClient().BlockRefByNumber(ctx, blockNum)
			require.NoError(t, err)
			blockA2, err := elA2.EthClient().BlockRefByNumber(ctx, blockNum)
			require.NoError(t, err)
			cancel()
			logger.Info("chain A", "blockNum", blockA.Number, "tip", blockA)
			logger.Info("chain A2", "blockNum", blockA2.Number, "tip", blockA2)
			return blockA, blockA2
		}

		targetBlockNum1 := uint64(10)
		logger.Info("wait until reaching target block", "blockNum", targetBlockNum1)
		require.Eventually(t, func() bool {
			blockA, blockA2 := queryLatest()
			return blockA.Number >= targetBlockNum1 && blockA2.Number >= targetBlockNum1
		}, 30*time.Second, waitTime)

		logger.Info("stop verifier")
		control.L2CLNodeState(ids.L2A2CL, stack.Stop)

		logger.Info("make sure verifier does not advance")
		var prevBlockA2 eth.BlockRef
		require.Eventually(t, func() bool {
			_, blockA2 := queryLatest()
			isStatic := prevBlockA2.Hash == blockA2.Hash
			prevBlockA2 = blockA2
			return isStatic
		}, 10*time.Second, waitTime)

		logger.Info("restart verifier")
		control.L2CLNodeState(ids.L2A2CL, stack.Start)

		logger.Info("explicit reconnection of L2CL P2P between sequencer and verifier")
		// wait until restarted L2CL can receive p2p API request
		time.Sleep(waitTime)
		WithL2CLP2PConnection(ids.L2ACL, ids.L2A2CL)(orch)

		targetBlockNum2 := uint64(30)
		require.Greater(t, targetBlockNum2, targetBlockNum1)
		logger.Info("wait until reaching target block", "blockNum", targetBlockNum2)
		require.Eventually(t, func() bool {
			blockA, blockA2 := queryLatest()
			return blockA.Number >= targetBlockNum2 && blockA2.Number >= targetBlockNum2
		}, 60*time.Second, waitTime)

		logger.Info("check sequencer and verifier holds identical chain until target block", "blockNum", targetBlockNum2)
		for blockNum := range targetBlockNum2 + 1 {
			blockA, blockA2 := queryBlock(blockNum)
			require.Equal(t, blockA.Hash, blockA2.Hash)
			require.Equal(t, blockNum, blockA.Number)
			require.Equal(t, blockNum, blockA2.Number)
			time.Sleep(50 * time.Millisecond)
		}
	}
}
