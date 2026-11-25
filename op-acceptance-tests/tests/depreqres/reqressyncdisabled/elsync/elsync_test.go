package elsync

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/depreqres/common"
	"github.com/ethereum-optimism/optimism/op-node/rollup/sync"
)

func TestUnsafeChainNotStalling_ELSync(gt *testing.T) {
	common.UnsafeChainNotStalling_Disconnect(gt, sync.ELSync, 20, common.ReqRespSyncDisabledOpts(sync.ELSync)...)
}

func TestUnsafeChainNotStalling_ELSync_RestartOpNode(gt *testing.T) {
	common.UnsafeChainNotStalling_RestartOpNode(gt, sync.ELSync, 20, common.ReqRespSyncDisabledOpts(sync.ELSync)...)
}
