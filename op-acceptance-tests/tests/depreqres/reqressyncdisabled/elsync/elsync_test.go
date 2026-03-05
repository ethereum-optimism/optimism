package elsync

import (
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/depreqres/common"
	bss "github.com/ethereum-optimism/optimism/op-batcher/batcher"
	"github.com/ethereum-optimism/optimism/op-devstack/compat"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-node/rollup/sync"
)

func systemOpts() []stack.CommonOption {
	return []stack.CommonOption{
		presets.WithExecutionLayerSyncOnVerifiers(),
		presets.WithCompatibleTypes(compat.SysGo),
		presets.WithReqRespSyncDisabled(),
		presets.WithNoDiscovery(),
		stack.MakeCommon(sysgo.WithBatcherOption(func(id stack.ComponentID, cfg *bss.CLIConfig) {
			cfg.Stopped = true
		})),
	}
}

func TestUnsafeChainNotStalling_ELSync_Short(gt *testing.T) {
	common.UnsafeChainNotStalling_Disconnect(gt, sync.ELSync, 20*time.Second, systemOpts()...)
}

func TestUnsafeChainNotStalling_ELSync_Long(gt *testing.T) {
	common.UnsafeChainNotStalling_Disconnect(gt, sync.ELSync, 95*time.Second, systemOpts()...)
}

func TestUnsafeChainNotStalling_ELSync_RestartOpNode_Long(gt *testing.T) {
	common.UnsafeChainNotStalling_RestartOpNode(gt, sync.ELSync, 95*time.Second, systemOpts()...)
}
