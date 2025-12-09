package sync

import (
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
)

const (
	// UnSafeHeadAdvanceRetries is the number of retries for unsafe head advancement
	UnSafeHeadAdvanceRetries = 15

	// CrossUnsafeHeadAdvanceRetries is the number of retries for cross-unsafe head advancement
	CrossUnsafeHeadAdvanceRetries = 15

	// LocalSafeHeadAdvanceRetries is the number of retries for safe head advancement
	LocalSafeHeadAdvanceRetries = 15

	// SafeHeadAdvanceRetries is the number of retries for safe head advancement
	SafeHeadAdvanceRetries = 25

	// FinalizedHeadAdvanceRetries is the number of retries for finalized head advancement
	FinalizedHeadAdvanceRetries = 100
)

func TestLocalUnsafeHeadAdvancing(gt *testing.T) {
	t := devtest.SerialT(gt)

	out := presets.NewSimpleInterop(t)
	l2aChainID := out.L2CLA.ChainID()
	l2bChainID := out.L2CLB.ChainID()

	supervisorStatus := out.Supervisor.FetchSyncStatus()

	out.Supervisor.WaitForL2HeadToAdvance(out.L2ChainA.ChainID(), 2, "unsafe", UnSafeHeadAdvanceRetries)
	out.Supervisor.WaitForL2HeadToAdvance(out.L2ChainB.ChainID(), 2, "unsafe", UnSafeHeadAdvanceRetries)

	// Wait and check if the local unsafe head has advanced on L2A
	err := wait.For(t.Ctx(), 2*time.Second, func() (bool, error) {
		status := out.L2CLA.SyncStatus()
		return status.UnsafeL2.Number > supervisorStatus.Chains[l2aChainID].LocalUnsafe.Number, nil
	})

	// Wait and check if the local unsafe head has advanced on L2B
	err = wait.For(t.Ctx(), 2*time.Second, func() (bool, error) {
		status := out.L2CLB.SyncStatus()
		return status.UnsafeL2.Number > supervisorStatus.Chains[l2bChainID].LocalUnsafe.Number, nil
	})

	// Wait and cross check the supervisor unsafe heads to advance on both chains
	err = wait.For(t.Ctx(), 5*time.Second, func() (bool, error) {
		latestSupervisorStatus := out.Supervisor.FetchSyncStatus()
		return latestSupervisorStatus.Chains[l2aChainID].LocalUnsafe.Number > supervisorStatus.Chains[l2aChainID].LocalUnsafe.Number &&
			latestSupervisorStatus.Chains[l2bChainID].LocalUnsafe.Number >= supervisorStatus.Chains[l2bChainID].LocalUnsafe.Number, nil
	})
	t.Require().NoError(err)
}

func TestCrossUnsafeHeadAdvancing(gt *testing.T) {
	t := devtest.SerialT(gt)

	out := presets.NewSimpleInterop(t)
	l2aChainID := out.L2CLA.ChainID()
	l2bChainID := out.L2CLB.ChainID()

	supervisorStatus := out.Supervisor.FetchSyncStatus()

	out.Supervisor.WaitForL2HeadToAdvance(out.L2ChainA.ChainID(), 2, "cross-unsafe", CrossUnsafeHeadAdvanceRetries)
	out.Supervisor.WaitForL2HeadToAdvance(out.L2ChainB.ChainID(), 2, "cross-unsafe", CrossUnsafeHeadAdvanceRetries)

	// Wait and cross check the supervisor cross unsafe heads to advance on both chains
	err := wait.For(t.Ctx(), 5*time.Second, func() (bool, error) {
		latestSupervisorStatus := out.Supervisor.FetchSyncStatus()
		return latestSupervisorStatus.Chains[l2aChainID].LocalUnsafe.Number > supervisorStatus.Chains[l2aChainID].LocalUnsafe.Number &&
			latestSupervisorStatus.Chains[l2bChainID].LocalUnsafe.Number >= supervisorStatus.Chains[l2bChainID].LocalUnsafe.Number, nil
	})

	// Wait and check if the cross unsafe head has advanced on L2A
	err = wait.For(t.Ctx(), 2*time.Second, func() (bool, error) {
		status := out.L2CLA.SyncStatus()
		return status.CrossUnsafeL2.Number > supervisorStatus.Chains[l2aChainID].CrossUnsafe.Number, nil
	})

	// Wait and check if the cross unsafe head has advanced on L2B
	err = wait.For(t.Ctx(), 2*time.Second, func() (bool, error) {
		status := out.L2CLB.SyncStatus()
		return status.CrossUnsafeL2.Number > supervisorStatus.Chains[l2bChainID].CrossUnsafe.Number, nil
	})

	t.Require().NoError(err)
}

func TestLocalSafeHeadAdvancing(gt *testing.T) {
	t := devtest.SerialT(gt)

	out := presets.NewSimpleInterop(t)
	l2aChainID := out.L2CLA.ChainID()
	l2bChainID := out.L2CLB.ChainID()

	supervisorStatus := out.Supervisor.FetchSyncStatus()

	out.Supervisor.WaitForL2HeadToAdvance(out.L2ChainA.ChainID(), 1, "local-safe", LocalSafeHeadAdvanceRetries)
	out.Supervisor.WaitForL2HeadToAdvance(out.L2ChainB.ChainID(), 1, "local-safe", LocalSafeHeadAdvanceRetries)

	// Wait and check if the local safe head has advanced on L2A
	err := wait.For(t.Ctx(), 2*time.Second, func() (bool, error) {
		status := out.L2CLA.SyncStatus()
		return status.LocalSafeL2.Number > supervisorStatus.Chains[l2aChainID].LocalSafe.Number, nil
	})

	// Wait and check if the local safe head has advanced on L2B
	err = wait.For(t.Ctx(), 2*time.Second, func() (bool, error) {
		status := out.L2CLB.SyncStatus()
		return status.LocalSafeL2.Number > supervisorStatus.Chains[l2bChainID].LocalSafe.Number, nil
	})

	// Wait and cross check the supervisor local safe heads to advance on both chains
	err = wait.For(t.Ctx(), 5*time.Second, func() (bool, error) {
		latestSupervisorStatus := out.Supervisor.FetchSyncStatus()
		return latestSupervisorStatus.Chains[l2aChainID].LocalSafe.Number > supervisorStatus.Chains[l2aChainID].LocalSafe.Number &&
			latestSupervisorStatus.Chains[l2bChainID].LocalSafe.Number >= supervisorStatus.Chains[l2bChainID].LocalSafe.Number, nil
	})
	t.Require().NoError(err)
}

func TestCrossSafeHeadAdvancing(gt *testing.T) {
	t := devtest.SerialT(gt)

	out := presets.NewSimpleInterop(t)
	l2aChainID := out.L2CLA.ChainID()
	l2bChainID := out.L2CLB.ChainID()

	supervisorStatus := out.Supervisor.FetchSyncStatus()

	out.Supervisor.WaitForL2HeadToAdvance(out.L2ChainA.ChainID(), 1, "safe", SafeHeadAdvanceRetries)
	out.Supervisor.WaitForL2HeadToAdvance(out.L2ChainB.ChainID(), 1, "safe", SafeHeadAdvanceRetries)

	// Wait and cross check the supervisor cross safe heads to advance on both chains
	err := wait.For(t.Ctx(), 5*time.Second, func() (bool, error) {
		latestSupervisorStatus := out.Supervisor.FetchSyncStatus()
		return latestSupervisorStatus.Chains[l2aChainID].CrossSafe.Number > supervisorStatus.Chains[l2aChainID].CrossSafe.Number &&
			latestSupervisorStatus.Chains[l2bChainID].CrossSafe.Number >= supervisorStatus.Chains[l2bChainID].CrossSafe.Number, nil
	})

	// Wait and check if the cross safe head has advanced on L2A
	err = wait.For(t.Ctx(), 2*time.Second, func() (bool, error) {
		status := out.L2CLA.SyncStatus()
		return status.SafeL2.Number > supervisorStatus.Chains[l2aChainID].CrossSafe.Number, nil
	})

	// Wait and check if the cross safe head has advanced on L2B
	err = wait.For(t.Ctx(), 2*time.Second, func() (bool, error) {
		status := out.L2CLB.SyncStatus()
		return status.SafeL2.Number > supervisorStatus.Chains[l2bChainID].CrossSafe.Number, nil
	})

	t.Require().NoError(err)
}

func TestMinSyncedL1Advancing(gt *testing.T) {
	t := devtest.SerialT(gt)

	out := presets.NewSimpleInterop(t)
	supervisorStatus := out.Supervisor.FetchSyncStatus()

	out.Supervisor.AwaitMinL1(supervisorStatus.MinSyncedL1.Number + 1)

	// Wait and check if the currentL1 head has advanced on L2A
	err := wait.For(t.Ctx(), 2*time.Second, func() (bool, error) {
		status := out.L2CLA.SyncStatus()
		return status.CurrentL1.Number > supervisorStatus.MinSyncedL1.Number, nil
	})

	// Wait and check if the currentL1 head has advanced on L2B
	err = wait.For(t.Ctx(), 2*time.Second, func() (bool, error) {
		status := out.L2CLB.SyncStatus()
		return status.CurrentL1.Number > supervisorStatus.MinSyncedL1.Number, nil
	})

	// Wait and check if the min synced L1 has advanced
	err = wait.For(t.Ctx(), 5*time.Second, func() (bool, error) {
		latestSupervisorStatus := out.Supervisor.FetchSyncStatus()
		return latestSupervisorStatus.MinSyncedL1.Number > supervisorStatus.MinSyncedL1.Number, nil
	})
	t.Require().NoError(err)
}

func TestFinalizedHeadAdvancing(gt *testing.T) {
	t := devtest.SerialT(gt)

	out := presets.NewSimpleInterop(t)
	l2aChainID := out.L2CLA.ChainID()
	l2bChainID := out.L2CLB.ChainID()

	supervisorStatus := out.Supervisor.FetchSyncStatus()

	out.Supervisor.WaitForL2HeadToAdvance(out.L2ChainA.ChainID(), 1, "finalized", FinalizedHeadAdvanceRetries)
	out.Supervisor.WaitForL2HeadToAdvance(out.L2ChainB.ChainID(), 1, "finalized", FinalizedHeadAdvanceRetries)

	// Wait and cross check the supervisor finalized heads to advance on both chains
	err := wait.For(t.Ctx(), 5*time.Second, func() (bool, error) {
		latestSupervisorStatus := out.Supervisor.FetchSyncStatus()
		return latestSupervisorStatus.Chains[l2aChainID].Finalized.Number > supervisorStatus.Chains[l2aChainID].Finalized.Number &&
			latestSupervisorStatus.Chains[l2bChainID].Finalized.Number >= supervisorStatus.Chains[l2bChainID].Finalized.Number, nil
	})

	// Wait and check if the finalized head has advanced on L2A
	err = wait.For(t.Ctx(), 2*time.Second, func() (bool, error) {
		status := out.L2CLA.SyncStatus()
		return status.FinalizedL1.Time > supervisorStatus.FinalizedTimestamp &&
			status.FinalizedL2.Number > supervisorStatus.Chains[l2aChainID].Finalized.Number, nil
	})

	// Wait and check if the finalized head has advanced on L2B
	err = wait.For(t.Ctx(), 2*time.Second, func() (bool, error) {
		status := out.L2CLB.SyncStatus()
		return status.FinalizedL1.Time > supervisorStatus.FinalizedTimestamp &&
			status.FinalizedL2.Number > supervisorStatus.Chains[l2bChainID].Finalized.Number, nil
	})
	t.Require().NoError(err)
}

func TestDerivationPipeline(gt *testing.T) {
	t := devtest.SerialT(gt)

	out := presets.NewSimpleInterop(t)
	l2BlockHead := out.Supervisor.L2HeadBlockID(out.L2ChainA.ChainID(), "local-safe")

	// Get current L1 at which L2 is at and wait for new L1 to be synced in supervisor.
	current_l1_at_l2 := out.L2CLA.SyncStatus().CurrentL1
	out.Supervisor.AwaitMinL1(current_l1_at_l2.Number + 1)
	new_l1 := out.Supervisor.FetchSyncStatus().MinSyncedL1

	t.Require().NotEqual(current_l1_at_l2.Hash, new_l1.Hash)
	t.Require().Greater(new_l1.Number, current_l1_at_l2.Number)

	//  Wait for the L2 chain to sync to the new L1 block.
	err := wait.For(t.Ctx(), 5*time.Second, func() (bool, error) {
		new_l1_at_l2 := out.L2CLA.SyncStatus().CurrentL1
		return new_l1_at_l2.Number >= new_l1.Number, nil
	})
	t.Require().NoError(err)

	new_l2BlockHead := out.Supervisor.L2HeadBlockID(out.L2ChainA.ChainID(), "local-safe")
	t.Require().Greater(new_l2BlockHead.Number, l2BlockHead.Number)
}
