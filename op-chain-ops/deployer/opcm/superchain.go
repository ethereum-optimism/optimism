package opcm

import (
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	opcrypto "github.com/ethereum-optimism/optimism/op-service/crypto"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
)

type DeploySuperchainInput struct {
	SuperchainProxyAdminOwner  common.Address         `toml:"superchainProxyAdminOwner"`
	ProtocolVersionsOwner      common.Address         `toml:"protocolVersionsOwner"`
	Guardian                   common.Address         `toml:"guardian"`
	Paused                     bool                   `toml:"paused"`
	RequiredProtocolVersion    params.ProtocolVersion `toml:"requiredProtocolVersion"`
	RecommendedProtocolVersion params.ProtocolVersion `toml:"recommendedProtocolVersion"`
}

func (dsi *DeploySuperchainInput) InputSet() bool {
	return true
}

type DeploySuperchainOutput struct {
	SuperchainProxyAdmin  common.Address
	SuperchainConfigImpl  common.Address
	SuperchainConfigProxy common.Address
	ProtocolVersionsImpl  common.Address
	ProtocolVersionsProxy common.Address
}

func (output *DeploySuperchainOutput) CheckOutput(input common.Address) error {
	return nil
}

type DeploySuperchainScript struct {
	Run func(in common.Address, out common.Address) error
}

type DeploySuperchainOpts struct {
	ChainID     *big.Int
	ArtifactsFS foundry.StatDirFs
	Deployer    common.Address
	Signer      opcrypto.SignerFn
	Input       DeploySuperchainInput
	Client      *ethclient.Client
	Logger      log.Logger
}

func DeploySuperchain(h *script.Host, input DeploySuperchainInput) (DeploySuperchainOutput, error) {
	var dso DeploySuperchainOutput

	inputAddr := h.NewScriptAddress()
	outputAddr := h.NewScriptAddress()

	cleanupInput, err := script.WithPrecompileAtAddress[*DeploySuperchainInput](h, inputAddr, &input)
	if err != nil {
		return dso, fmt.Errorf("failed to insert DeploySuperchainInput precompile: %w", err)
	}
	defer cleanupInput()

	cleanupOutput, err := script.WithPrecompileAtAddress[*DeploySuperchainOutput](
		h,
		outputAddr,
		&dso,
		script.WithFieldSetter[*DeploySuperchainOutput],
	)
	if err != nil {
		return dso, fmt.Errorf("failed to insert DeploySuperchainOutput precompile: %w", err)
	}
	defer cleanupOutput()

	deployScript, cleanupDeploy, err := script.WithScript[DeploySuperchainScript](h, "DeploySuperchain.s.sol", "DeploySuperchain")
	if err != nil {
		return dso, fmt.Errorf("failed to load DeploySuperchain script: %w", err)
	}
	defer cleanupDeploy()

	if err := deployScript.Run(inputAddr, outputAddr); err != nil {
		return dso, fmt.Errorf("failed to run DeploySuperchain script: %w", err)
	}

	return dso, nil
}
