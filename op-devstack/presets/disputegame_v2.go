package presets

import (
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
)

func WithDisputeGameV2() stack.CommonOption {
	return stack.MakeCommon(sysgo.WithDeployerOptions(sysgo.WithDevFeatureBitmap(deployer.DeployV2DisputeGamesDevFlag)))
}
