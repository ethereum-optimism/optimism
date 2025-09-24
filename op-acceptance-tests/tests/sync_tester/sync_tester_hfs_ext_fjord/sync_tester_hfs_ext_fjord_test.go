package sync_tester_hfs_ext_fjord

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/sync_tester/hardforks_ext"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/sync"
)

func TestSyncTesterHFS_Fjord_CLSync(gt *testing.T) {
	hardforks_ext.SyncTesterHFSExt(gt, rollup.Fjord, sync.CLSync)
}

func TestSyncTesterHFS_Fjord_ELSync(gt *testing.T) {
	hardforks_ext.SyncTesterHFSExt(gt, rollup.Fjord, sync.ELSync)
}
