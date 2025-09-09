package sync_tester_hfs_ext_fjord

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/sync_tester/hardforks_ext"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
)

func TestSyncTesterHFS_Fjord(gt *testing.T) {
	forkTimestamp := func(net *dsl.L2Network) *uint64 {
		return net.Escape().RollupConfig().FjordTime
	}
	hardforks_ext.SyncTesterHFSExt(gt, "Fjord", forkTimestamp)
}
