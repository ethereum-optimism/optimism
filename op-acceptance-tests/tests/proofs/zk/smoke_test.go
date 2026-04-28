package zk

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
)

// TestSmoke verifies that op-deployer correctly deploys and registers the ZK
// dispute game when ZKDisputeGameFlag is enabled.
func TestSmoke(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := newSystem(t)
	require := t.Require()

	zk := sys.DisputeGameFactory().ZKGameImpl()

	require.NotEmpty(zk.Address, "ZK dispute game impl must be registered in DisputeGameFactory")
	require.NotZero(zk.Args.MaxChallengeDuration, "maxChallengeDuration must be set")
	require.NotZero(zk.Args.MaxProveDuration, "maxProveDuration must be set")
	require.Positive(zk.Args.ChallengerBond.Sign(), "challengerBond must be non-zero")
	require.Equal(sys.L2Chain.ChainID().ToBig(), zk.Args.L2ChainID, "l2ChainId must match deployed chain")
}
