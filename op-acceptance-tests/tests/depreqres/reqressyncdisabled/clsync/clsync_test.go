package clsync

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/depreqres/common"
	"github.com/ethereum-optimism/optimism/op-node/rollup/sync"
)

func TestUnsafeChainNotStalling_CLSync_Short(gt *testing.T) {
	common.UnsafeChainNotStalling_Disconnect(gt, sync.CLSync, 10, common.ReqRespSyncDisabledOpts(sync.CLSync)...)
}

func TestUnsafeChainNotStalling_CLSync_Long(gt *testing.T) {
	common.UnsafeChainNotStalling_Disconnect(gt, sync.CLSync, 47, common.ReqRespSyncDisabledOpts(sync.CLSync)...)
}

func TestUnsafeChainNotStalling_CLSync_RestartOpNode_Long(gt *testing.T) {
	common.UnsafeChainNotStalling_RestartOpNode(gt, sync.CLSync, 47, common.ReqRespSyncDisabledOpts(sync.CLSync)...)
}
