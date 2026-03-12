package elsync

import (
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/depreqres/common"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-node/rollup/sync"
)

const syncModeReqRespSyncFlakyReason = "known flaky in the default acceptance run"

func TestUnsafeChainNotStalling_ELSync_Short(gt *testing.T) {
	t := devtest.ParallelT(gt)
	t.MarkFlaky(syncModeReqRespSyncFlakyReason)
	common.UnsafeChainNotStalling_DisconnectT(t, sync.ELSync, 20*time.Second, common.SyncModeReqRespSyncOpts(sync.ELSync)...)
}

func TestUnsafeChainNotStalling_ELSync_Long(gt *testing.T) {
	t := devtest.ParallelT(gt)
	t.MarkFlaky(syncModeReqRespSyncFlakyReason)
	common.UnsafeChainNotStalling_DisconnectT(t, sync.ELSync, 95*time.Second, common.SyncModeReqRespSyncOpts(sync.ELSync)...)
}

func TestUnsafeChainNotStalling_ELSync_RestartOpNode_Long(gt *testing.T) {
	t := devtest.ParallelT(gt)
	t.MarkFlaky(syncModeReqRespSyncFlakyReason)
	common.UnsafeChainNotStalling_RestartOpNodeT(t, sync.ELSync, 95*time.Second, common.SyncModeReqRespSyncOpts(sync.ELSync)...)
}
