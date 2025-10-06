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

	sys := presets.NewSimpleWithSyncTester(t)

	// Sanity: both CLs should advance to a small target using their respective EL backends
	// L2CL: regular EL; L2CL2: SyncTester-provided EL (unique session)
	target := uint64(12)
	dsl.CheckAll(t,
		sys.L2CL.AdvancedFn(types.LocalUnsafe, target, 60),
		sys.L2CL2.AdvancedFn(types.LocalUnsafe, target, 60),
	)

	// Cross-check: both CLs unsafe heads must exist on the read-only EL
	head1 := sys.L2CL.SyncStatus().UnsafeL2
	head2 := sys.L2CL2.SyncStatus().UnsafeL2

	require.GreaterOrEqual(head1.Number, target)
	require.GreaterOrEqual(head2.Number, target)
	require.Equal(sys.L2EL.BlockRefByNumber(head1.Number).Hash, head1.Hash)
	require.Equal(sys.L2EL.BlockRefByNumber(head2.Number).Hash, head2.Hash)
}
