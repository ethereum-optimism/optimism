package sync_tester_hfs_ext_canyon

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/sync_tester/hardforks_ext"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
)

func TestSyncTesterHFS_Canyon(gt *testing.T) {
	forkTimestamp := func(net *dsl.L2Network) *uint64 {
		return net.Escape().ChainConfig().CanyonTime
	}
	hardforks_ext.SyncTesterHFSExt(gt, "Canyon", forkTimestamp)
}
