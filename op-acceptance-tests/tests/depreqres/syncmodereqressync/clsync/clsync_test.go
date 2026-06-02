package clsync

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/depreqres/common"
	"github.com/ethereum-optimism/optimism/op-node/rollup/sync"
)

func TestUnsafeChainNotStalling_CLSync(gt *testing.T) {
	common.UnsafeChainNotStalling_Disconnect(gt, sync.CLSync, 20, common.SyncModeReqRespSyncOpts(sync.CLSync)...)
}

func TestUnsafeChainNotStalling_CLSync_RestartOpNode(gt *testing.T) {
	common.UnsafeChainNotStalling_RestartOpNode(gt, sync.CLSync, 20, common.SyncModeReqRespSyncOpts(sync.CLSync)...)
}
