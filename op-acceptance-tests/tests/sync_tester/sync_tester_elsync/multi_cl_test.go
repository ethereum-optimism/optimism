package sync_tester_elsync

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

// Validates multiple CLs can sync concurrently when backed by a single Sync Tester
// Each CL initializes its own mock EL endpoint (session) via the Sync Tester service.
func TestSyncTester_MultipleCLsSingleSyncTester(gt *testing.T) {
	t := devtest.SerialT(gt)
	require := t.Require()

	// Use a system with two CLs, each backed by its own SyncTester EL endpoint
	sys := presets.NewMultiWithSyncTester(t)

	// Sanity: both CLs should advance to a small target using their respective SyncTester EL backends (unique sessions)
	target := uint64(12)
	dsl.CheckAll(t,
		sys.L2CL_A.AdvancedFn(types.LocalUnsafe, target, 60),
		sys.L2CL_B.AdvancedFn(types.LocalUnsafe, target, 60),
	)

	// Cross-check: both CLs' unsafe heads must exist on their corresponding SyncTester-backed ELs
	head1 := sys.L2CL_A.SyncStatus().UnsafeL2
	head2 := sys.L2CL_B.SyncStatus().UnsafeL2

	require.GreaterOrEqual(head1.Number, target)
	require.GreaterOrEqual(head2.Number, target)
	require.Equal(sys.SyncTesterL2ELA.BlockRefByNumber(head1.Number).Hash, head1.Hash)
	require.Equal(sys.SyncTesterL2ELB.BlockRefByNumber(head2.Number).Hash, head2.Hash)
}
