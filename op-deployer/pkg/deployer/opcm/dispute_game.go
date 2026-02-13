package opcm

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/forge"
)

type DeployDisputeGameInput struct {
	Release                  string
	GameKind                 string
	GameType                 uint32
	AbsolutePrestate         common.Hash
	MaxGameDepth             *big.Int
	SplitDepth               *big.Int
	ClockExtension           uint64
	MaxClockDuration         uint64
	DelayedWethProxy         common.Address
	AnchorStateRegistryProxy common.Address
	VmAddress                common.Address
	L2ChainId                *big.Int
	Proposer                 common.Address
	Challenger               common.Address
}

type DeployDisputeGameOutput struct {
	DisputeGameImpl common.Address
}

type DeployDisputeGameScript script.DeployScriptWithOutput[DeployDisputeGameInput, DeployDisputeGameOutput]

// NewDeployDisputeGameScript loads and validates the DeployDisputeGame2 script contract
func NewDeployDisputeGameScript(host *script.Host) (DeployDisputeGameScript, error) {
	return script.NewDeployScriptWithOutputFromFile[DeployDisputeGameInput, DeployDisputeGameOutput](host, "DeployDisputeGame.s.sol", "DeployDisputeGame")
}

func NewDeployDisputeGameForgeCaller(client *forge.Client) forge.ScriptCaller[DeployDisputeGameInput, DeployDisputeGameOutput] {
	return forge.NewScriptCaller(
		client,
		"scripts/deploy/DeployDisputeGame.s.sol:DeployDisputeGame",
		"runWithBytes(bytes)",
		&forge.BytesScriptEncoder[DeployDisputeGameInput]{TypeName: "DeployDisputeGameInput"},
		&forge.BytesScriptDecoder[DeployDisputeGameOutput]{TypeName: "DeployDisputeGameOutput"},
	)
}

// DeployDisputeGameViaForge deploys dispute game contracts using Forge
func DeployDisputeGameViaForge(env *ForgeEnv, input DeployDisputeGameInput) (DeployDisputeGameOutput, error) {
	var output DeployDisputeGameOutput
	if err := env.validate(true); err != nil {
		return output, err
	}
	forgeCaller := NewDeployDisputeGameForgeCaller(env.Client)
	var err error
	output, _, err = forgeCaller(env.Context, input, env.buildForgeOpts()...)
	if err != nil {
		return output, fmt.Errorf("failed to deploy dispute game with Forge: %w", err)
	}
	return output, nil
}
