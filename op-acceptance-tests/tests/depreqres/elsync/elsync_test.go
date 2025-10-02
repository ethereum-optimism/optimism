package elsync

import (
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/depreqres/common"
	"github.com/ethereum-optimism/optimism/op-node/rollup/sync"
)

func TestUnsafeChainStalling_ELSync_Short(gt *testing.T) {
	common.UnsafeChainStalling_DisabledReqRespSync(gt, sync.ELSync, 20*time.Second)
}

func TestUnsafeChainStalling_ELSync_Long(gt *testing.T) {
	common.UnsafeChainStalling_DisabledReqRespSync(gt, sync.ELSync, 185*time.Second)
}

func TestUnsafeChainStalling_ELSync_RestartOpNode_Long(gt *testing.T) {
	common.UnsafeChainStalling_DisabledReqRespSync_RestartOpNode(gt, sync.ELSync, 185*time.Second)
}
