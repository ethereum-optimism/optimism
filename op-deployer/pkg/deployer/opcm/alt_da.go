package opcm

import (
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/forge"
	"github.com/ethereum/go-ethereum/common"
)

type DeployAltDAInput struct {
	Salt                     common.Hash
	ProxyAdmin               common.Address
	ChallengeContractOwner   common.Address
	ChallengeWindow          *big.Int
	ResolveWindow            *big.Int
	BondSize                 *big.Int
	ResolverRefundPercentage *big.Int
}

type DeployAltDAOutput struct {
	DataAvailabilityChallengeProxy common.Address
	DataAvailabilityChallengeImpl  common.Address
}

type DeployAltDAScript script.DeployScriptWithOutput[DeployAltDAInput, DeployAltDAOutput]

// NewDeployAltDAScript loads and validates the DeployAltDA script contract
func NewDeployAltDAScript(host *script.Host) (DeployAltDAScript, error) {
	return script.NewDeployScriptWithOutputFromFile[DeployAltDAInput, DeployAltDAOutput](host, "DeployAltDA.s.sol", "DeployAltDA")
}

func NewDeployAltDAForgeCaller(client *forge.Client) forge.ScriptCaller[DeployAltDAInput, DeployAltDAOutput] {
	return forge.NewScriptCaller(
		client,
		"scripts/deploy/DeployAltDA.s.sol:DeployAltDA",
		"runWithBytes(bytes)",
		&forge.BytesScriptEncoder[DeployAltDAInput]{TypeName: "DeployAltDAInput"},
		&forge.BytesScriptDecoder[DeployAltDAOutput]{TypeName: "DeployAltDAOutput"},
	)
}

// DeployAltDAViaForge deploys AltDA contracts using Forge
func DeployAltDAViaForge(env *ForgeEnv, input DeployAltDAInput) (DeployAltDAOutput, error) {
	var output DeployAltDAOutput
	if err := env.validate(true); err != nil {
		return output, err
	}
	forgeCaller := NewDeployAltDAForgeCaller(env.Client)
	var err error
	output, _, err = forgeCaller(env.Context, input, env.buildForgeOpts()...)
	if err != nil {
		return output, fmt.Errorf("failed to deploy alt-da contracts with Forge: %w", err)
	}
	return output, nil
}
