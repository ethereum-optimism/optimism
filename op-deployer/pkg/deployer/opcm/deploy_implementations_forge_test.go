package opcm

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestDeployImplementationsForgeEncoder(t *testing.T) {
	encoder := &DeployImplementationsForgeEncoder{}

	input := DeployImplementationsForgeInput{
		WithdrawalDelaySeconds:          big.NewInt(604800),
		MinProposalSizeBytes:            big.NewInt(126000),
		ChallengePeriodSeconds:          big.NewInt(86400),
		ProofMaturityDelaySeconds:       big.NewInt(604800),
		DisputeGameFinalityDelaySeconds: big.NewInt(302400),
		MipsVersion:                     big.NewInt(1),
		SuperchainConfigProxy:           common.HexToAddress("0x1234567890123456789012345678901234567890"),
		ProtocolVersionsProxy:           common.HexToAddress("0x2345678901234567890123456789012345678901"),
		SuperchainProxyAdmin:            common.HexToAddress("0x3456789012345678901234567890123456789012"),
		UpgradeController:               common.HexToAddress("0x4567890123456789012345678901234567890123"),
		Challenger:                      common.HexToAddress("0x5678901234567890123456789012345678901234"),
	}

	encoded, err := encoder.Encode(input)
	require.NoError(t, err)
	require.NotEmpty(t, encoded)

	// Verify the encoded data has the expected length
	// 11 fields: 6 uint256 (32 bytes each) + 5 addresses (32 bytes each) = 352 bytes
	require.Equal(t, 352, len(encoded))
}

func TestDeployImplementationsForgeDecoder(t *testing.T) {
	decoder := &DeployImplementationsForgeDecoder{}

	// Test that decoder handles invalid input gracefully
	_, err := decoder.Decode([]byte{})
	require.Error(t, err)

	// Test that decoder handles random bytes gracefully
	_, err = decoder.Decode([]byte{1, 2, 3, 4})
	require.Error(t, err)
}

func TestConvertToForgeInput(t *testing.T) {
	opcmInput := DeployImplementationsInput{
		WithdrawalDelaySeconds:          big.NewInt(604800),
		MinProposalSizeBytes:            big.NewInt(126000),
		ChallengePeriodSeconds:          big.NewInt(86400),
		ProofMaturityDelaySeconds:       big.NewInt(604800),
		DisputeGameFinalityDelaySeconds: big.NewInt(302400),
		MipsVersion:                     big.NewInt(1),
		SuperchainConfigProxy:           common.HexToAddress("0x1234567890123456789012345678901234567890"),
		ProtocolVersionsProxy:           common.HexToAddress("0x2345678901234567890123456789012345678901"),
		SuperchainProxyAdmin:            common.HexToAddress("0x3456789012345678901234567890123456789012"),
		UpgradeController:               common.HexToAddress("0x4567890123456789012345678901234567890123"),
		Challenger:                      common.HexToAddress("0x5678901234567890123456789012345678901234"),
	}

	forgeInput := ConvertToForgeInput(opcmInput)

	require.Equal(t, opcmInput.WithdrawalDelaySeconds, forgeInput.WithdrawalDelaySeconds)
	require.Equal(t, opcmInput.MinProposalSizeBytes, forgeInput.MinProposalSizeBytes)
	require.Equal(t, opcmInput.ChallengePeriodSeconds, forgeInput.ChallengePeriodSeconds)
	require.Equal(t, opcmInput.ProofMaturityDelaySeconds, forgeInput.ProofMaturityDelaySeconds)
	require.Equal(t, opcmInput.DisputeGameFinalityDelaySeconds, forgeInput.DisputeGameFinalityDelaySeconds)
	require.Equal(t, opcmInput.MipsVersion, forgeInput.MipsVersion)
	require.Equal(t, opcmInput.SuperchainConfigProxy, forgeInput.SuperchainConfigProxy)
	require.Equal(t, opcmInput.ProtocolVersionsProxy, forgeInput.ProtocolVersionsProxy)
	require.Equal(t, opcmInput.SuperchainProxyAdmin, forgeInput.SuperchainProxyAdmin)
	require.Equal(t, opcmInput.UpgradeController, forgeInput.UpgradeController)
	require.Equal(t, opcmInput.Challenger, forgeInput.Challenger)
}

func TestConvertFromForgeOutput(t *testing.T) {
	forgeOutput := DeployImplementationsForgeOutput{
		Opcm:                             common.HexToAddress("0x1111111111111111111111111111111111111111"),
		OpcmContractsContainer:           common.HexToAddress("0x2222222222222222222222222222222222222222"),
		OpcmGameTypeAdder:                common.HexToAddress("0x3333333333333333333333333333333333333333"),
		OpcmDeployer:                     common.HexToAddress("0x4444444444444444444444444444444444444444"),
		OpcmUpgrader:                     common.HexToAddress("0x5555555555555555555555555555555555555555"),
		OpcmInteropMigrator:              common.HexToAddress("0x6666666666666666666666666666666666666666"),
		OpcmStandardValidator:            common.HexToAddress("0x7777777777777777777777777777777777777777"),
		DelayedWETHImpl:                  common.HexToAddress("0x8888888888888888888888888888888888888888"),
		OptimismPortalImpl:               common.HexToAddress("0x9999999999999999999999999999999999999999"),
		ETHLockboxImpl:                   common.HexToAddress("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
		PreimageOracleSingleton:          common.HexToAddress("0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"),
		MipsSingleton:                    common.HexToAddress("0xcccccccccccccccccccccccccccccccccccccccc"),
		SystemConfigImpl:                 common.HexToAddress("0xdddddddddddddddddddddddddddddddddddddddd"),
		L1CrossDomainMessengerImpl:       common.HexToAddress("0xeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"),
		L1ERC721BridgeImpl:               common.HexToAddress("0xffffffffffffffffffffffffffffffffffffffff"),
		L1StandardBridgeImpl:             common.HexToAddress("0x0000000000000000000000000000000000000001"),
		OptimismMintableERC20FactoryImpl: common.HexToAddress("0x0000000000000000000000000000000000000002"),
		DisputeGameFactoryImpl:           common.HexToAddress("0x0000000000000000000000000000000000000003"),
		AnchorStateRegistryImpl:          common.HexToAddress("0x0000000000000000000000000000000000000004"),
		SuperchainConfigImpl:             common.HexToAddress("0x0000000000000000000000000000000000000005"),
		ProtocolVersionsImpl:             common.HexToAddress("0x0000000000000000000000000000000000000006"),
	}

	opcmOutput := ConvertFromForgeOutput(forgeOutput)

	require.Equal(t, forgeOutput.Opcm, opcmOutput.Opcm)
	require.Equal(t, forgeOutput.OpcmContractsContainer, opcmOutput.OpcmContractsContainer)
	require.Equal(t, forgeOutput.OpcmGameTypeAdder, opcmOutput.OpcmGameTypeAdder)
	require.Equal(t, forgeOutput.OpcmDeployer, opcmOutput.OpcmDeployer)
	require.Equal(t, forgeOutput.OpcmUpgrader, opcmOutput.OpcmUpgrader)
	require.Equal(t, forgeOutput.OpcmInteropMigrator, opcmOutput.OpcmInteropMigrator)
	require.Equal(t, forgeOutput.OpcmStandardValidator, opcmOutput.OpcmStandardValidator)
	require.Equal(t, forgeOutput.DelayedWETHImpl, opcmOutput.DelayedWETHImpl)
	require.Equal(t, forgeOutput.OptimismPortalImpl, opcmOutput.OptimismPortalImpl)
	require.Equal(t, forgeOutput.ETHLockboxImpl, opcmOutput.ETHLockboxImpl)
	require.Equal(t, forgeOutput.PreimageOracleSingleton, opcmOutput.PreimageOracleSingleton)
	require.Equal(t, forgeOutput.MipsSingleton, opcmOutput.MipsSingleton)
	require.Equal(t, forgeOutput.SystemConfigImpl, opcmOutput.SystemConfigImpl)
	require.Equal(t, forgeOutput.L1CrossDomainMessengerImpl, opcmOutput.L1CrossDomainMessengerImpl)
	require.Equal(t, forgeOutput.L1ERC721BridgeImpl, opcmOutput.L1ERC721BridgeImpl)
	require.Equal(t, forgeOutput.L1StandardBridgeImpl, opcmOutput.L1StandardBridgeImpl)
	require.Equal(t, forgeOutput.OptimismMintableERC20FactoryImpl, opcmOutput.OptimismMintableERC20FactoryImpl)
	require.Equal(t, forgeOutput.DisputeGameFactoryImpl, opcmOutput.DisputeGameFactoryImpl)
	require.Equal(t, forgeOutput.AnchorStateRegistryImpl, opcmOutput.AnchorStateRegistryImpl)
	require.Equal(t, forgeOutput.SuperchainConfigImpl, opcmOutput.SuperchainConfigImpl)
	require.Equal(t, forgeOutput.ProtocolVersionsImpl, opcmOutput.ProtocolVersionsImpl)
}
