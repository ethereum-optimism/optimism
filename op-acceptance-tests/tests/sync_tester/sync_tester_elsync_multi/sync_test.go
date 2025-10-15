package multi

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

func TestMultiELSync(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewSimpleWithSyncTester(t)
	require := t.Require()

	// Stop L2CL2 to control SyncTesterL2EL manually
	sys.L2CL2.Stop()

	// Advance few blocks to make sure reference node advanced
	sys.L2CL.Advanced(types.LocalUnsafe, 10, 30)

	// Manually trigger multiple EL Syncs using the engine API
	startNum := sys.SyncTesterL2EL.BlockRefByLabel(eth.Unsafe).Number

	// Sync Tester will track all unsafe payloads which can not be appended to unsafe head
	// Sync Tester will use the WindowSyncPolicy, and mocks completion of EL Sync when the consecutive
	// payloads were previously observed

	targetNum := startNum + 5
	// Invoke consecutive FCU calls to mock L2CL2 behavior:
	// L2CL2 will unsafe payloads from the sequencer and do newPayload, FCU call
	// L2CL2 will push consecutive unsafe payloads, assuming sequencer sent payloads arrived in order

	// First Run of EL Sync: Scenario: Mock op-node with consecutive newPayload + FCU calls
	// Start with targetNum-2=startNum + 3 to trigger EL Sync. SyncTesterEL unsafe head=startNum
	sys.SyncTesterL2EL.NewPayload(sys.L2EL, targetNum-2).IsSyncing()
	sys.SyncTesterL2EL.ForkchoiceUpdate(sys.L2EL, targetNum-2, 0, 0, nil).IsSyncing()
	// Window filled: [startNum+3]

	sys.SyncTesterL2EL.NewPayload(sys.L2EL, targetNum-1).IsSyncing()
	sys.SyncTesterL2EL.ForkchoiceUpdate(sys.L2EL, targetNum-1, 0, 0, nil).IsSyncing()
	// Window filled: [startNum+3,startNum+4]

	// Consecutive window size(two) payloads were FCUed
	// Sync Tester will mock by advancing the non canonical chain to make the next newPayload return VALID
	sys.SyncTesterL2EL.NewPayload(sys.L2EL, targetNum).IsValid()
	sys.SyncTesterL2EL.ForkchoiceUpdate(sys.L2EL, targetNum, 0, 0, nil).IsValid()
	// EL Sync Completed to targetNum

	require.Equal(targetNum, sys.L2EL.BlockRefByHash(sys.SyncTesterL2EL.BlockRefByLabel(eth.Unsafe).Hash).Number)

	// Second Run of EL Sync: Scenario: Manaul FCU, different with engine API pattern
	// Start with targetNum-2=startNum+8 to trigger EL Sync. SyncTesterEL unsafe head=startNum+5
	targetNum = startNum + 10
	sys.SyncTesterL2EL.NewPayload(sys.L2EL, targetNum-2).IsSyncing()
	sys.SyncTesterL2EL.ForkchoiceUpdate(sys.L2EL, targetNum-2, 0, 0, nil).IsSyncing()
	// Window filled: [targetNum+8]

	sys.SyncTesterL2EL.NewPayload(sys.L2EL, targetNum-1).IsSyncing()
	sys.SyncTesterL2EL.ForkchoiceUpdate(sys.L2EL, targetNum-1, 0, 0, nil).IsSyncing()
	// Window filled: [startNum+8,startNum+9]

	// Consecutive window size(two) payloads were FCUed
	// Sync Tester will mock by advancing the non canonical chain to make the next newPayload return VALID
	// If we call FCU targeting startNum again, it will return VALID, canonicalizing
	sys.SyncTesterL2EL.ForkchoiceUpdate(sys.L2EL, targetNum-1, 0, 0, nil).IsValid()

	// Next call with newPayload, FCU will also VALID because the payload may be directly appended to the unsafe head
	sys.SyncTesterL2EL.NewPayload(sys.L2EL, targetNum).IsValid()
	sys.SyncTesterL2EL.ForkchoiceUpdate(sys.L2EL, targetNum, 0, 0, nil).IsValid()

	require.Equal(targetNum, sys.L2EL.BlockRefByHash(sys.SyncTesterL2EL.BlockRefByLabel(eth.Unsafe).Hash).Number)

	t.Cleanup(func() {
		sys.L2CL2.Start()
	})
}
