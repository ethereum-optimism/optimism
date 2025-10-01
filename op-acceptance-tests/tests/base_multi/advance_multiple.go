package base_multi

import (
	"os"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/stretchr/testify/require"
)

// TestCLAdvanceMultiple verifies two L2 chains advance when using separate CLs
func TestCLAdvanceMultiple(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewTwoL2(t)

	blockTime := sys.L2A.Escape().RollupConfig().BlockTime
	waitTime := time.Duration(blockTime+1) * time.Second

	// Check L2A advances
	numA := sys.L2ACL.SyncStatus().UnsafeL2.Number
	newA := numA
	require.Eventually(t, func() bool {
		newA, numA = sys.L2ACL.SyncStatus().UnsafeL2.Number, newA
		return newA > numA
	}, 30*time.Second, waitTime)

	// Check L2B advances
	numB := sys.L2BCL.SyncStatus().UnsafeL2.Number
	newB := numB
	require.Eventually(t, func() bool {
		newB, numB = sys.L2BCL.SyncStatus().UnsafeL2.Number, newB
		return newB > numB
	}, 30*time.Second, waitTime)
}

// TestCLAdvanceMultiple_Supernode verifies two L2 chains advance when using a single shared supernode CL.
func TestCLAdvanceMultiple_Supernode(gt *testing.T) {
	prev := os.Getenv("DEVSTACK_L2CL_KIND")
	_ = os.Setenv("DEVSTACK_L2CL_KIND", "supernode")
	defer func() { _ = os.Setenv("DEVSTACK_L2CL_KIND", prev) }()

	TestCLAdvanceMultiple(gt)
}
