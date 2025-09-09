package sync_tester_hfs_ext_ecotone

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/sync_tester/hardforks_ext"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
)

func TestSyncTesterHFS_Ecotone(gt *testing.T) {
	forkTimestamp := func(net *dsl.L2Network) *uint64 {
		return net.Escape().ChainConfig().EcotoneTime
	}
	hardforks_ext.SyncTesterHFSExt(gt, "Ecotone", forkTimestamp)
}
