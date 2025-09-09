package sync_tester_hfs_ext_holocene

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/sync_tester/hardforks_ext"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
)

func TestSyncTesterHFS_Holocene(gt *testing.T) {
	forkTimestamp := func(net *dsl.L2Network) *uint64 {
		return net.Escape().RollupConfig().HoloceneTime
	}
	hardforks_ext.SyncTesterHFSExt(gt, "Holocene", forkTimestamp)
}
