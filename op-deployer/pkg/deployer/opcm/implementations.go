package opcm

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
)

type DeployImplementationsInput struct {
	WithdrawalDelaySeconds          *big.Int
	MinProposalSizeBytes            *big.Int
	ChallengePeriodSeconds          *big.Int
	ProofMaturityDelaySeconds       *big.Int
	DisputeGameFinalityDelaySeconds *big.Int
	MipsVersion                     *big.Int
	// Release version to set OPCM implementations for, of the format `op-contracts/vX.Y.Z`.
	L1ContractsRelease    string
	SuperchainConfigProxy common.Address
	ProtocolVersionsProxy common.Address
	SuperchainProxyAdmin  common.Address
	UpgradeController     common.Address
	UseInterop            bool // if true, deploy Interop implementations
}

func (input *DeployImplementationsInput) InputSet() bool {
	return true
}

type DeployImplementationsOutput struct {
	Opcm                             common.Address `json:"opcmAddress"`
	OpcmContractsContainer           common.Address `json:"opcmContractsContainerAddress"`
	OpcmGameTypeAdder                common.Address `json:"opcmGameTypeAdderAddress"`
	OpcmDeployer                     common.Address `json:"opcmDeployerAddress"`
	OpcmUpgrader                     common.Address `json:"opcmUpgraderAddress"`
	DelayedWETHImpl                  common.Address `json:"delayedWETHImplAddress"`
	OptimismPortalImpl               common.Address `json:"optimismPortalImplAddress"`
	ETHLockboxImpl                   common.Address `json:"ethLockboxImplAddress" evm:"ethLockboxImpl"`
	PreimageOracleSingleton          common.Address `json:"preimageOracleSingletonAddress"`
	MipsSingleton                    common.Address `json:"mipsSingletonAddress"`
	SystemConfigImpl                 common.Address `json:"systemConfigImplAddress"`
	L1CrossDomainMessengerImpl       common.Address `json:"l1CrossDomainMessengerImplAddress"`
	L1ERC721BridgeImpl               common.Address `json:"l1ERC721BridgeImplAddress"`
	L1StandardBridgeImpl             common.Address `json:"l1StandardBridgeImplAddress"`
	OptimismMintableERC20FactoryImpl common.Address `json:"optimismMintableERC20FactoryImplAddress"`
	DisputeGameFactoryImpl           common.Address `json:"disputeGameFactoryImplAddress"`
	AnchorStateRegistryImpl          common.Address `json:"anchorStateRegistryImplAddress"`
	SuperchainConfigImpl             common.Address `json:"superchainConfigImplAddress"`
	ProtocolVersionsImpl             common.Address `json:"protocolVersionsImplAddress"`
}

func (output *DeployImplementationsOutput) CheckOutput(input common.Address) error {
	return nil
}

type DeployImplementationsScript struct {
	Run func(input, output common.Address) error
}

func DeployImplementations(
	host *script.Host,
	input DeployImplementationsInput,
) (DeployImplementationsOutput, error) {
	var output DeployImplementationsOutput
	inputAddr := host.NewScriptAddress()
	outputAddr := host.NewScriptAddress()

	cleanupInput, err := script.WithPrecompileAtAddress[*DeployImplementationsInput](host, inputAddr, &input)
	if err != nil {
		return output, fmt.Errorf("failed to insert DeployImplementationsInput precompile: %w", err)
	}
	defer cleanupInput()

	cleanupOutput, err := script.WithPrecompileAtAddress[*DeployImplementationsOutput](host, outputAddr, &output,
		script.WithFieldSetter[*DeployImplementationsOutput])
	if err != nil {
		return output, fmt.Errorf("failed to insert DeployImplementationsOutput precompile: %w", err)
	}
	defer cleanupOutput()

	implContract := "DeployImplementations"
	deployScript, cleanupDeploy, err := script.WithScript[DeployImplementationsScript](host, "DeployImplementations.s.sol", implContract)
	if err != nil {
		return output, fmt.Errorf("failed to load %s script: %w", implContract, err)
	}
	defer cleanupDeploy()

	opcmContract := "OPContractsManager"
	if err := host.RememberOnLabel("OPContractsManager", opcmContract+".sol", opcmContract); err != nil {
		return output, fmt.Errorf("failed to link OPContractsManager label: %w", err)
	}

	// So we can see in detail where the SystemConfig interop initializer fails
	sysConfig := "SystemConfig"
	if err := host.RememberOnLabel("SystemConfigImpl", sysConfig+".sol", sysConfig); err != nil {
		return output, fmt.Errorf("failed to link SystemConfig label: %w", err)
	}

	if err := deployScript.Run(inputAddr, outputAddr); err != nil {
		return output, fmt.Errorf("failed to run %s script: %w", implContract, err)
	}

	return output, nil
}
