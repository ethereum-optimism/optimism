package sync_tester_hfs_ext_delta

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/sync_tester/hardforks_ext"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/sync"
)

func TestSyncTesterHFS_Delta_CLSync(gt *testing.T) {
	hardforks_ext.SyncTesterHFSExt(gt, rollup.Delta, sync.CLSync)
}

func TestSyncTesterHFS_Delta_ELSync(gt *testing.T) {
	hardforks_ext.SyncTesterHFSExt(gt, rollup.Delta, sync.ELSync)
}
