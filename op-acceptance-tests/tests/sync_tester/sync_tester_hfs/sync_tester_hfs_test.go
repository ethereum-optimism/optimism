package sync_tester_hfs

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

func TestSyncTesterHardforks(gt *testing.T) {
	t := devtest.SerialT(gt)

	sys := presets.NewSimpleWithSyncTester(t)
	require := t.Require()

	// Hardforks will be activated from Bedrock to Jovian, 10 hardforks with 6 second time delta between.
	// 6 * 10 = 60s , so we need at least 30 (60 / 2 + 1) L2 blocks with block time 2 to make the CL experience scheduled hardforks.
	targetNum := 32
	dsl.CheckAll(t,
		sys.L2CL.AdvancedFn(types.LocalUnsafe, uint64(targetNum), targetNum+10),
		sys.L2CL2.AdvancedFn(types.LocalUnsafe, uint64(targetNum), targetNum+10),
	)

	current := sys.L2CL2.HeadBlockRef(types.LocalUnsafe)

	// Check the L2CL passed configured hardforks
	jovianTime := sys.L2Chain.Escape().ChainConfig().JovianTime
	require.NotNil(jovianTime, "jovian must be activated")
	require.Greater(current.Time, *jovianTime, "must pass jovian block")
	// Check block hash state from L2CL2 which was synced using the sync tester
	require.Equal(sys.L2EL.BlockRefByNumber(current.Number).Hash, current.Hash, "hash mismatch")
}
