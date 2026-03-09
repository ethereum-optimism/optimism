package throttling

import (
	"testing"
	"time"

	bss "github.com/ethereum-optimism/optimism/op-batcher/batcher"
	batcherConfig "github.com/ethereum-optimism/optimism/op-batcher/config"
	"github.com/ethereum-optimism/optimism/op-devstack/compat"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
)

const blockSizeLimit = 5_000

func TestMain(m *testing.M) {
	presets.DoMain(m,
		presets.WithMinimal(),
		presets.WithCompatibleTypes(compat.SysGo),
		stack.MakeCommon(sysgo.WithBatcherOption(func(id stack.ComponentID, cfg *bss.CLIConfig) {
			// Enable throttling with step controller for predictable behavior
			cfg.ThrottleConfig.LowerThreshold = 99 // > 0 enables the throttling loop.
			cfg.ThrottleConfig.UpperThreshold = 100
			cfg.ThrottleConfig.ControllerType = batcherConfig.StepControllerType

			cfg.ThrottleConfig.BlockSizeLowerLimit = blockSizeLimit - 1
			cfg.ThrottleConfig.BlockSizeUpperLimit = blockSizeLimit

			cfg.PollInterval = 500 * time.Millisecond // Fast poll for quicker test feedback
		})),
	)
}
