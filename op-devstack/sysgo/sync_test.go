package sysgo

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/shim"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

// TestL2CLSyncP2P checks that unsafe head is propagated from sequencer to verifier.
// Tests started/restarted L2CL advances unsafe head via P2P connection.
func TestL2CLSyncP2P(gt *testing.T) {
	var ids DefaultRedundancyInteropSystemIDs
	opt := DefaultRedundancyInteropSystem(&ids)

	logger := testlog.Logger(gt, log.LevelInfo)

	p := devtest.NewP(context.Background(), logger, func() {
		gt.Helper()
		gt.FailNow()
	})
	gt.Cleanup(p.Close)

	orch := NewOrchestrator(p)
	stack.ApplyOptionLifecycle(opt, orch)

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

		WithL2CLP2PConnection(ids.L2ACL, ids.L2A2CL).AfterDeploy(orch)

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

// TestUnsafeChainUnknownToL2CL tests the below scenario:
// supervisor unsafe ahead of L2CL unsafe, aka L2CL processes new blocks first.
// To create this out-of-sync scenario, we follow the steps below:
// 1. Make sequencer (L2CL), verifier (L2CL), and supervisor sync for a few blocks.
// - Sequencer and verifier are connected via P2P, which makes their unsafe heads in sync.
// - Both L2CLs are in managed mode, digesting L1 blocks from the supervisor and reporting unsafe and safe blocks back to the supervisor.
// 2. Disconnect the P2P connection between the sequencer and verifier.
// - The verifier will not receive unsafe heads via P2P, and can only update unsafe heads matching with safe heads by reading L1 batches.
// - The verifier safe head will lag behind or match the sequencer and supervisor because all three components share the same L1 view.
// - The verifier unsafe head will lag and always be out of sync compared to the sequencer and supervisor.
// - The supervisor unsafe head is ahead of the verifier unsafe head, which means there are unsafe blocks unknown to the verifier.
// 3. Reconnect the P2P connection between the sequencer and verifier.
// - The sequencer will broadcast all unknown unsafe blocks to the verifier.
// - The verifier will quickly catch up with the sequencer unsafe head as well as the supervisor.
// - The verifier will process previously unknown unsafe blocks and advance its unsafe head.
func TestUnsafeChainUnknownToL2CL(gt *testing.T) {
	var ids DefaultRedundancyInteropSystemIDs
	opt := DefaultRedundancyInteropSystem(&ids)

	logger := testlog.Logger(gt, log.LevelInfo)

	p := devtest.NewP(context.Background(), logger, func() {
		gt.Helper()
		gt.FailNow()
	})
	gt.Cleanup(p.Close)

	orch := NewOrchestrator(p)
	stack.ApplyOptionLifecycle(opt, orch)

	t := devtest.SerialT(gt)
	system := shim.NewSystem(t)
	orch.Hydrate(system)

	blockTime := system.L2Network(ids.L2A).RollupConfig().BlockTime

	waitTime := time.Duration(blockTime+1) * time.Second
	{
		logger := system.T().Logger()

		elA := system.L2Network(ids.L2A).L2ELNode(ids.L2AEL)
		elA2 := system.L2Network(ids.L2A).L2ELNode(ids.L2A2EL)
		clA := system.L2Network(ids.L2A).L2CLNode(ids.L2ACL)
		clA2 := system.L2Network(ids.L2A).L2CLNode(ids.L2A2CL)
		supervisor := system.Supervisor(ids.Supervisor)

		queryEL := func(label eth.BlockLabel) (eth.BlockRef, eth.BlockRef) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
			blockA, err := elA.EthClient().BlockRefByLabel(ctx, label)
			require.NoError(t, err)
			blockA2, err := elA2.EthClient().BlockRefByLabel(ctx, label)
			require.NoError(t, err)
			cancel()
			logger.Info("chain A", "blockNum", blockA.Number, "block", blockA)
			logger.Info("chain A2", "blockNum", blockA2.Number, "block", blockA2)
			return blockA, blockA2
		}

		queryCL := func() (*eth.SyncStatus, *eth.SyncStatus) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
			syncA, err := clA.RollupAPI().SyncStatus(ctx)
			require.NoError(t, err)
			syncA2, err := clA2.RollupAPI().SyncStatus(ctx)
			require.NoError(t, err)
			cancel()
			logger.Info("chain A", "sync", syncA)
			logger.Info("chain A2", "sync", syncA2)
			return syncA, syncA2
		}

		querySupervisor := func(chainID eth.ChainID) eth.BlockID {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
			view, err := supervisor.QueryAPI().LocalUnsafe(ctx, chainID)
			require.NoError(t, err)
			cancel()
			return view
		}

		targetBlockNum1 := uint64(10)
		logger.Info("wait until reaching target block", "blockNum", targetBlockNum1)
		require.Eventually(t, func() bool {
			blockA, blockA2 := queryEL(eth.Unsafe)
			return blockA.Number >= targetBlockNum1 && blockA2.Number >= targetBlockNum1
		}, 30*time.Second, waitTime)

		logger.Info("disconnect p2p between L2CLs")
		DisconnectL2CLP2P(ids.L2ACL, ids.L2A2CL).AfterDeploy(orch)

		// verifier lost its P2P connection with sequencer, and will advance its unsafe head by reading L1 but not by P2P
		_, prevblockA2 := queryEL(eth.Unsafe)
		targetBlockNum2 := prevblockA2.Number + 5
		logger.Info("make sure verifier advances safe head by reading L1", "blockNum", targetBlockNum2)
		require.Eventually(t, func() bool {
			_, syncA2 := queryCL()
			// unsafe head and safe head both advanced from last observed unsafe head
			return syncA2.SafeL2.Number == syncA2.UnsafeL2.Number && syncA2.SafeL2.Number > targetBlockNum2
		}, 60*time.Second, waitTime)

		logger.Info("verifier heads will lag compared from sequencer heads and supervisor view")
		require.Never(t, func() bool {
			syncA, syncA2 := queryCL()
			chainAView := querySupervisor(elA2.ChainID())
			// unsafe head will always lagged
			check := syncA.UnsafeL2.Number > syncA2.UnsafeL2.Number
			// safe head may be matched or lagged
			check = check && syncA.SafeL2.Number >= syncA2.SafeL2.Number
			// unsafe head may be matched or lagged compared to supervisor unsafe head view for chain A
			check = check && chainAView.Number >= syncA2.UnsafeL2.Number
			logger.Info("unsafe head sync status", "sequencer", syncA.UnsafeL2.Number, "supervisor", chainAView.Number, "verifier", syncA2.UnsafeL2.Number)
			return !check
		}, 15*time.Second, waitTime)

		logger.Info("explicit reconnection of L2CL P2P between sequencer and verifier")
		WithL2CLP2PConnection(ids.L2ACL, ids.L2A2CL).AfterDeploy(orch)

		logger.Info("verifier catchs up sequencer unsafe chain with was unknown for verifier")
		require.Eventually(t, func() bool {
			blockA, blockA2 := queryEL(eth.Unsafe)
			return blockA.Number == blockA2.Number && blockA.Hash == blockA2.Hash
		}, 10*time.Second, waitTime)
	}
}
