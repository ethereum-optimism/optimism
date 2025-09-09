package sync_tester_hfs_ext_granite

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/sync_tester/hardforks_ext"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
)

func TestSyncTesterHFS_Granite(gt *testing.T) {
	forkTimestamp := func(net *dsl.L2Network) *uint64 {
		return net.Escape().ChainConfig().GraniteTime
	}
	hardforks_ext.SyncTesterHFSExt(gt, "Granite", forkTimestamp)
}
