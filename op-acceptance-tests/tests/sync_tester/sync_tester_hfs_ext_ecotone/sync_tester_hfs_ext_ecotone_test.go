package sync_tester_hfs_ext_ecotone

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/sync_tester/hardforks_ext"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
)

func TestSyncTesterHFS_Ecotone(gt *testing.T) {
	hardforks_ext.SyncTesterHFSExt(gt, rollup.Ecotone)
}
