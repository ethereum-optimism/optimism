package elsync

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/depreqres/common"
	"github.com/ethereum-optimism/optimism/op-node/rollup/sync"
)

func TestUnsafeChainNotStalling_ELSync_Short(gt *testing.T) {
	common.UnsafeChainNotStalling_Disconnect(gt, sync.ELSync, 10, common.SyncModeReqRespSyncOpts(sync.ELSync)...)
}

func TestUnsafeChainNotStalling_ELSync_Long(gt *testing.T) {
	common.UnsafeChainNotStalling_Disconnect(gt, sync.ELSync, 47, common.SyncModeReqRespSyncOpts(sync.ELSync)...)
}

func TestUnsafeChainNotStalling_ELSync_RestartOpNode_Long(gt *testing.T) {
	common.UnsafeChainNotStalling_RestartOpNode(gt, sync.ELSync, 47, common.SyncModeReqRespSyncOpts(sync.ELSync)...)
}
