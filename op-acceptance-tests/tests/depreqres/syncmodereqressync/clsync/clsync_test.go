package clsync

import (
	"testing"
	"time"

	bss "github.com/ethereum-optimism/optimism/op-batcher/batcher"
	"github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/depreqres/common"
	"github.com/ethereum-optimism/optimism/op-devstack/compat"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-node/rollup/sync"
)

func systemOpts() []stack.CommonOption {
	return []stack.CommonOption{
		presets.WithConsensusLayerSync(),
		presets.WithCompatibleTypes(compat.SysGo),
		presets.WithSyncModeReqRespSync(),
		presets.WithNoDiscovery(),
		stack.MakeCommon(sysgo.WithBatcherOption(func(id stack.L2BatcherID, cfg *bss.CLIConfig) {
			cfg.Stopped = true
		})),
	}
}

func TestUnsafeChainNotStalling_CLSync_Short(gt *testing.T) {
	common.UnsafeChainNotStalling_Disconnect(gt, sync.CLSync, 20*time.Second, systemOpts()...)
}

func TestUnsafeChainNotStalling_CLSync_Long(gt *testing.T) {
	common.UnsafeChainNotStalling_Disconnect(gt, sync.CLSync, 95*time.Second, systemOpts()...)
}

func TestUnsafeChainNotStalling_CLSync_RestartOpNode_Long(gt *testing.T) {
	common.UnsafeChainNotStalling_RestartOpNode(gt, sync.CLSync, 95*time.Second, systemOpts()...)
}
