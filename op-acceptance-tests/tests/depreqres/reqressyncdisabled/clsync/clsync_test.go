package clsync

import (
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/depreqres/common"
	"github.com/ethereum-optimism/optimism/op-node/rollup/sync"
)

func TestUnsafeChainStalling_CLSync_Short(gt *testing.T) {
	common.UnsafeChainNotStalling_Disconnect(gt, sync.CLSync, 20*time.Second)
}

func TestUnsafeChainStalling_CLSync_Long(gt *testing.T) {
	common.UnsafeChainNotStalling_Disconnect(gt, sync.CLSync, 185*time.Second)
}

func TestUnsafeChainStalling_CLSync_RestartOpNode_Long(gt *testing.T) {
	common.UnsafeChainNotStalling_RestartOpNode(gt, sync.CLSync, 185*time.Second)
}
