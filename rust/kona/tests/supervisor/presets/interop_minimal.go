package presets

import (
	"os"
	"path/filepath"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/artifacts"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	oppresets "github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/intentbuilder"
)

func NewSimpleInteropMinimal(t devtest.T, opts ...oppresets.Option) *oppresets.SimpleInterop {
	artifactsPath := os.Getenv("OP_DEPLOYER_ARTIFACTS")
	if artifactsPath == "" {
		panic("OP_DEPLOYER_ARTIFACTS is not set")
	}
	contractArtifacts := artifacts.MustNewFileLocator(filepath.Join(artifactsPath, "src"))

	baseOpts := []oppresets.Option{
		oppresets.WithDeployerOptions(
			func(_ devtest.T, _ devkeys.Keys, builder intentbuilder.Builder) {
				builder.WithL1ContractsLocator(contractArtifacts)
				builder.WithL2ContractsLocator(contractArtifacts)
			},
		),
	}
	baseOpts = append(baseOpts, opts...)
	return oppresets.NewSimpleInterop(t, baseOpts...)
}

func WithSuggestedInteropActivationOffset(offset uint64) oppresets.Option {
	return oppresets.WithSuggestedInteropActivationOffset(offset)
}

func WithInteropNotAtGenesis() oppresets.Option {
	return oppresets.WithInteropNotAtGenesis()
}
