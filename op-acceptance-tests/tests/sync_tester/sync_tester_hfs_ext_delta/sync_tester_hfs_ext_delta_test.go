package sync_tester_hfs_ext_delta

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/sync_tester/hardforks_ext"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
)

func TestSyncTesterHFS_Delta(gt *testing.T) {
	forkTimestamp := func(net *dsl.L2Network) *uint64 {
		return net.Escape().RollupConfig().DeltaTime
	}
	hardforks_ext.SyncTesterHFSExt(gt, "Delta", forkTimestamp)
}
