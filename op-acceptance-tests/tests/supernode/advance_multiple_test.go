package supernode

import (
	"net/url"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/stretchr/testify/require"
)

// TestCLAdvanceMultiple verifies two L2 chains advance when using a shared CL
// it confirms:
// - the two L2 chains are on different chains
// - the two CLs are using the same supernode
// - the two CLs are advancing
func TestCLAdvanceMultiple(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewTwoL2(t)

	blockTime := sys.L2A.Escape().RollupConfig().BlockTime
	waitTime := time.Duration(blockTime+1) * time.Second

	// Check L2A advances
	numA := sys.L2ACL.SyncStatus().UnsafeL2.Number
	numB := sys.L2BCL.SyncStatus().UnsafeL2.Number

	// Check that the two CLs are on different chains
	require.NotEqual(t, sys.L2ACL.ChainID(), sys.L2BCL.ChainID())

	// Check that the two CLs are using the same supernode
	uA, err := url.Parse(sys.L2ACL.Escape().UserRPC())
	require.NoError(t, err)
	uB, err := url.Parse(sys.L2BCL.Escape().UserRPC())
	require.NoError(t, err)
	require.Equal(t, uA.Scheme, uB.Scheme)
	require.Equal(t, uA.Host, uB.Host)

	require.Eventually(t, func() bool {
		newA := sys.L2ACL.SyncStatus().UnsafeL2.Number
		newB := sys.L2BCL.SyncStatus().UnsafeL2.Number
		return newA > numA && newB > numB
	}, 30*time.Second, waitTime)

}
