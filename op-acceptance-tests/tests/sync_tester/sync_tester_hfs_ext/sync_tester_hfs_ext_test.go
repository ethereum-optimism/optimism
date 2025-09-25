package sync_tester_hfs_ext

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/sync_tester/hardforks_ext"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/sync"
)

func TestSyncTesterHFS_Canyon_CLSync(gt *testing.T) {
	hardforks_ext.SyncTesterHFSExt(gt, rollup.Canyon, sync.CLSync)
}

func TestSyncTesterHFS_Canyon_ELSync(gt *testing.T) {
	hardforks_ext.SyncTesterHFSExt(gt, rollup.Canyon, sync.ELSync)
}

func TestSyncTesterHFS_Delta_CLSync(gt *testing.T) {
	hardforks_ext.SyncTesterHFSExt(gt, rollup.Delta, sync.CLSync)
}

func TestSyncTesterHFS_Delta_ELSync(gt *testing.T) {
	hardforks_ext.SyncTesterHFSExt(gt, rollup.Delta, sync.ELSync)
}

func TestSyncTesterHFS_Ecotone_CLSync(gt *testing.T) {
	hardforks_ext.SyncTesterHFSExt(gt, rollup.Ecotone, sync.CLSync)
}

func TestSyncTesterHFS_Ecotone_ELSync(gt *testing.T) {
	hardforks_ext.SyncTesterHFSExt(gt, rollup.Ecotone, sync.ELSync)
}

func TestSyncTesterHFS_Fjord_CLSync(gt *testing.T) {
	hardforks_ext.SyncTesterHFSExt(gt, rollup.Fjord, sync.CLSync)
}

func TestSyncTesterHFS_Fjord_ELSync(gt *testing.T) {
	hardforks_ext.SyncTesterHFSExt(gt, rollup.Fjord, sync.ELSync)
}

func TestSyncTesterHFS_Granite_CLSync(gt *testing.T) {
	hardforks_ext.SyncTesterHFSExt(gt, rollup.Granite, sync.CLSync)
}

func TestSyncTesterHFS_Granite_ELSync(gt *testing.T) {
	hardforks_ext.SyncTesterHFSExt(gt, rollup.Granite, sync.ELSync)
}

func TestSyncTesterHFS_Holocene_CLSync(gt *testing.T) {
	hardforks_ext.SyncTesterHFSExt(gt, rollup.Holocene, sync.CLSync)
}

func TestSyncTesterHFS_Holocene_ELSync(gt *testing.T) {
	hardforks_ext.SyncTesterHFSExt(gt, rollup.Holocene, sync.ELSync)
}

func TestSyncTesterHFS_Isthmus_CLSync(gt *testing.T) {
	hardforks_ext.SyncTesterHFSExt(gt, rollup.Isthmus, sync.CLSync)
}

func TestSyncTesterHFS_Isthmus_ELSync(gt *testing.T) {
	hardforks_ext.SyncTesterHFSExt(gt, rollup.Isthmus, sync.ELSync)
}
