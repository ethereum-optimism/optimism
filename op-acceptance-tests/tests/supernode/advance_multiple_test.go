package supernode

import (
	"context"
	"net/url"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/chain_container/engine_controller"
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
	unsafeA := sys.L2ACL.SyncStatus().UnsafeL2
	numA := unsafeA.Number
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

	// Check L2A advances
	unsafeA = sys.L2ACL.SyncStatus().UnsafeL2
	require.Greater(t, unsafeA.Number, uint64(0)) // should be block 1, TODO assert this
	unsafeATime := unsafeA.Time
	numA = unsafeA.Number

	sys.L2A.WaitForBlock()

	unsafeA = sys.L2ACL.SyncStatus().UnsafeL2
	require.Greater(t, unsafeA.Number, uint64(1))

	// create a supernode engine controller, and use that to
	// rewind one of the chains
	ec := sys.L2A.L2ELNodes()[0].Escape().L2EngineClient()
	ed := sys.L2A.L2ELNodes()[0].Escape().L2EthClient()
	type foo struct {
		apis.EngineClient
		apis.L2EthClient
	}
	engineController := engine_controller.NewEngineControllerWithL2AndRollup(foo{ec, ed}, sys.L2A.Escape().RollupConfig())

	////   _______  [synthetic]
	//    /
	// [0] <- [1] <- [2]
	// sf             u
	//
	// The method FCUs to synthetic first, and then back to 1.
	err = engineController.RewindToTimestamp(context.Background(), unsafeATime)
	require.NoError(t, err)

	// Check reset works as expected
	resetUnsafe, err := sys.L2A.L2ELNodes()[0].Escape().L2EthClient().BlockRefByLabel(context.Background(), eth.Unsafe)
	require.NoError(t, err)
	require.Less(t, resetUnsafe.Number, unsafeA.Number)
}
