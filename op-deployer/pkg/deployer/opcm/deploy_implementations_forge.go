package opcm

import (
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/forge"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

// DeployImplementationsForgeInput represents the Forge script input structure
type DeployImplementationsForgeInput struct {
	WithdrawalDelaySeconds          *big.Int
	MinProposalSizeBytes            *big.Int
	ChallengePeriodSeconds          *big.Int
	ProofMaturityDelaySeconds       *big.Int
	DisputeGameFinalityDelaySeconds *big.Int
	MipsVersion                     *big.Int
	SuperchainConfigProxy           common.Address
	ProtocolVersionsProxy           common.Address
	SuperchainProxyAdmin            common.Address
	UpgradeController               common.Address
	Challenger                      common.Address
}

// DeployImplementationsForgeOutput represents the Forge script output structure
type DeployImplementationsForgeOutput struct {
	Opcm                             common.Address
	OpcmContractsContainer           common.Address
	OpcmGameTypeAdder                common.Address
	OpcmDeployer                     common.Address
	OpcmUpgrader                     common.Address
	OpcmInteropMigrator              common.Address
	OpcmStandardValidator            common.Address
	DelayedWETHImpl                  common.Address
	OptimismPortalImpl               common.Address
	ETHLockboxImpl                   common.Address
	PreimageOracleSingleton          common.Address
	MipsSingleton                    common.Address
	SystemConfigImpl                 common.Address
	L1CrossDomainMessengerImpl       common.Address
	L1ERC721BridgeImpl               common.Address
	L1StandardBridgeImpl             common.Address
	OptimismMintableERC20FactoryImpl common.Address
	DisputeGameFactoryImpl           common.Address
	AnchorStateRegistryImpl          common.Address
	SuperchainConfigImpl             common.Address
	ProtocolVersionsImpl             common.Address
}

// DeployImplementationsForgeEncoder implements forge.ScriptCallEncoder
type DeployImplementationsForgeEncoder struct{}

func (e *DeployImplementationsForgeEncoder) Encode(input DeployImplementationsForgeInput) ([]byte, error) {
	// Define the ABI type for the Input struct
	inputType, err := abi.NewType("tuple", "DeployImplementations.Input", []abi.ArgumentMarshaling{
		{Name: "withdrawalDelaySeconds", Type: "uint256"},
		{Name: "minProposalSizeBytes", Type: "uint256"},
		{Name: "challengePeriodSeconds", Type: "uint256"},
		{Name: "proofMaturityDelaySeconds", Type: "uint256"},
		{Name: "disputeGameFinalityDelaySeconds", Type: "uint256"},
		{Name: "mipsVersion", Type: "uint256"},
		{Name: "superchainConfigProxy", Type: "address"},
		{Name: "protocolVersionsProxy", Type: "address"},
		{Name: "superchainProxyAdmin", Type: "address"},
		{Name: "upgradeController", Type: "address"},
		{Name: "challenger", Type: "address"},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create input type: %w", err)
	}

	args := abi.Arguments{{Type: inputType}}

	// Pack the input struct
	encoded, err := args.Pack(struct {
		WithdrawalDelaySeconds          *big.Int
		MinProposalSizeBytes            *big.Int
		ChallengePeriodSeconds          *big.Int
		ProofMaturityDelaySeconds       *big.Int
		DisputeGameFinalityDelaySeconds *big.Int
		MipsVersion                     *big.Int
		SuperchainConfigProxy           common.Address
		ProtocolVersionsProxy           common.Address
		SuperchainProxyAdmin            common.Address
		UpgradeController               common.Address
		Challenger                      common.Address
	}{
		WithdrawalDelaySeconds:          input.WithdrawalDelaySeconds,
		MinProposalSizeBytes:            input.MinProposalSizeBytes,
		ChallengePeriodSeconds:          input.ChallengePeriodSeconds,
		ProofMaturityDelaySeconds:       input.ProofMaturityDelaySeconds,
		DisputeGameFinalityDelaySeconds: input.DisputeGameFinalityDelaySeconds,
		MipsVersion:                     input.MipsVersion,
		SuperchainConfigProxy:           input.SuperchainConfigProxy,
		ProtocolVersionsProxy:           input.ProtocolVersionsProxy,
		SuperchainProxyAdmin:            input.SuperchainProxyAdmin,
		UpgradeController:               input.UpgradeController,
		Challenger:                      input.Challenger,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to encode input: %w", err)
	}

	return encoded, nil
}

// DeployImplementationsForgeDecoder implements forge.ScriptCallDecoder
type DeployImplementationsForgeDecoder struct{}

func (d *DeployImplementationsForgeDecoder) Decode(raw []byte) (DeployImplementationsForgeOutput, error) {
	// Define the ABI type for the Output struct
	outputType, err := abi.NewType("tuple", "DeployImplementations.Output", []abi.ArgumentMarshaling{
		{Name: "opcm", Type: "address"},
		{Name: "opcmContractsContainer", Type: "address"},
		{Name: "opcmGameTypeAdder", Type: "address"},
		{Name: "opcmDeployer", Type: "address"},
		{Name: "opcmUpgrader", Type: "address"},
		{Name: "opcmInteropMigrator", Type: "address"},
		{Name: "opcmStandardValidator", Type: "address"},
		{Name: "delayedWETHImpl", Type: "address"},
		{Name: "optimismPortalImpl", Type: "address"},
		{Name: "ethLockboxImpl", Type: "address"},
		{Name: "preimageOracleSingleton", Type: "address"},
		{Name: "mipsSingleton", Type: "address"},
		{Name: "systemConfigImpl", Type: "address"},
		{Name: "l1CrossDomainMessengerImpl", Type: "address"},
		{Name: "l1ERC721BridgeImpl", Type: "address"},
		{Name: "l1StandardBridgeImpl", Type: "address"},
		{Name: "optimismMintableERC20FactoryImpl", Type: "address"},
		{Name: "disputeGameFactoryImpl", Type: "address"},
		{Name: "anchorStateRegistryImpl", Type: "address"},
		{Name: "superchainConfigImpl", Type: "address"},
		{Name: "protocolVersionsImpl", Type: "address"},
	})
	if err != nil {
		return DeployImplementationsForgeOutput{}, fmt.Errorf("failed to create output type: %w", err)
	}

	args := abi.Arguments{{Type: outputType}}

	// Unpack the output
	unpacked, err := args.Unpack(raw)
	if err != nil {
		return DeployImplementationsForgeOutput{}, fmt.Errorf("failed to unpack output: %w", err)
	}

	if len(unpacked) != 1 {
		return DeployImplementationsForgeOutput{}, fmt.Errorf("expected 1 unpacked value, got %d", len(unpacked))
	}

	// Convert to struct
	result := unpacked[0].(struct {
		Opcm                             common.Address
		OpcmContractsContainer           common.Address
		OpcmGameTypeAdder                common.Address
		OpcmDeployer                     common.Address
		OpcmUpgrader                     common.Address
		OpcmInteropMigrator              common.Address
		OpcmStandardValidator            common.Address
		DelayedWETHImpl                  common.Address
		OptimismPortalImpl               common.Address
		ETHLockboxImpl                   common.Address
		PreimageOracleSingleton          common.Address
		MipsSingleton                    common.Address
		SystemConfigImpl                 common.Address
		L1CrossDomainMessengerImpl       common.Address
		L1ERC721BridgeImpl               common.Address
		L1StandardBridgeImpl             common.Address
		OptimismMintableERC20FactoryImpl common.Address
		DisputeGameFactoryImpl           common.Address
		AnchorStateRegistryImpl          common.Address
		SuperchainConfigImpl             common.Address
		ProtocolVersionsImpl             common.Address
	})

	return DeployImplementationsForgeOutput{
		Opcm:                             result.Opcm,
		OpcmContractsContainer:           result.OpcmContractsContainer,
		OpcmGameTypeAdder:                result.OpcmGameTypeAdder,
		OpcmDeployer:                     result.OpcmDeployer,
		OpcmUpgrader:                     result.OpcmUpgrader,
		OpcmInteropMigrator:              result.OpcmInteropMigrator,
		OpcmStandardValidator:            result.OpcmStandardValidator,
		DelayedWETHImpl:                  result.DelayedWETHImpl,
		OptimismPortalImpl:               result.OptimismPortalImpl,
		ETHLockboxImpl:                   result.ETHLockboxImpl,
		PreimageOracleSingleton:          result.PreimageOracleSingleton,
		MipsSingleton:                    result.MipsSingleton,
		SystemConfigImpl:                 result.SystemConfigImpl,
		L1CrossDomainMessengerImpl:       result.L1CrossDomainMessengerImpl,
		L1ERC721BridgeImpl:               result.L1ERC721BridgeImpl,
		L1StandardBridgeImpl:             result.L1StandardBridgeImpl,
		OptimismMintableERC20FactoryImpl: result.OptimismMintableERC20FactoryImpl,
		DisputeGameFactoryImpl:           result.DisputeGameFactoryImpl,
		AnchorStateRegistryImpl:          result.AnchorStateRegistryImpl,
		SuperchainConfigImpl:             result.SuperchainConfigImpl,
		ProtocolVersionsImpl:             result.ProtocolVersionsImpl,
	}, nil
}

// NewDeployImplementationsForgeCaller creates a new forge.ScriptCaller for DeployImplementations
func NewDeployImplementationsForgeCaller(client *forge.Client) forge.ScriptCaller[DeployImplementationsForgeInput, DeployImplementationsForgeOutput] {
	encoder := &DeployImplementationsForgeEncoder{}
	decoder := &DeployImplementationsForgeDecoder{}

	return forge.NewScriptCaller[DeployImplementationsForgeInput, DeployImplementationsForgeOutput](
		client,
		"scripts/deploy/DeployImplementations.s.sol:DeployImplementations",
		"run((uint256,uint256,uint256,uint256,uint256,uint256,address,address,address,address,address))",
		encoder,
		decoder,
	)
}

// ConvertToForgeInput converts DeployImplementationsInput to DeployImplementationsForgeInput
func ConvertToForgeInput(input DeployImplementationsInput) DeployImplementationsForgeInput {
	return DeployImplementationsForgeInput{
		WithdrawalDelaySeconds:          input.WithdrawalDelaySeconds,
		MinProposalSizeBytes:            input.MinProposalSizeBytes,
		ChallengePeriodSeconds:          input.ChallengePeriodSeconds,
		ProofMaturityDelaySeconds:       input.ProofMaturityDelaySeconds,
		DisputeGameFinalityDelaySeconds: input.DisputeGameFinalityDelaySeconds,
		MipsVersion:                     input.MipsVersion,
		SuperchainConfigProxy:           input.SuperchainConfigProxy,
		ProtocolVersionsProxy:           input.ProtocolVersionsProxy,
		SuperchainProxyAdmin:            input.SuperchainProxyAdmin,
		UpgradeController:               input.UpgradeController,
		Challenger:                      input.Challenger,
	}
}

// ConvertFromForgeOutput converts DeployImplementationsForgeOutput to DeployImplementationsOutput
func ConvertFromForgeOutput(output DeployImplementationsForgeOutput) DeployImplementationsOutput {
	return DeployImplementationsOutput{
		Opcm:                             output.Opcm,
		OpcmContractsContainer:           output.OpcmContractsContainer,
		OpcmGameTypeAdder:                output.OpcmGameTypeAdder,
		OpcmDeployer:                     output.OpcmDeployer,
		OpcmUpgrader:                     output.OpcmUpgrader,
		OpcmInteropMigrator:              output.OpcmInteropMigrator,
		OpcmStandardValidator:            output.OpcmStandardValidator,
		DelayedWETHImpl:                  output.DelayedWETHImpl,
		OptimismPortalImpl:               output.OptimismPortalImpl,
		ETHLockboxImpl:                   output.ETHLockboxImpl,
		PreimageOracleSingleton:          output.PreimageOracleSingleton,
		MipsSingleton:                    output.MipsSingleton,
		SystemConfigImpl:                 output.SystemConfigImpl,
		L1CrossDomainMessengerImpl:       output.L1CrossDomainMessengerImpl,
		L1ERC721BridgeImpl:               output.L1ERC721BridgeImpl,
		L1StandardBridgeImpl:             output.L1StandardBridgeImpl,
		OptimismMintableERC20FactoryImpl: output.OptimismMintableERC20FactoryImpl,
		DisputeGameFactoryImpl:           output.DisputeGameFactoryImpl,
		AnchorStateRegistryImpl:          output.AnchorStateRegistryImpl,
		SuperchainConfigImpl:             output.SuperchainConfigImpl,
		ProtocolVersionsImpl:             output.ProtocolVersionsImpl,
	}
}
