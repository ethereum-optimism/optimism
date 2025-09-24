package standard

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/ethereum-optimism/superchain-registry/validation"

	"github.com/ethereum/go-ethereum/superchain"

	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	op_service "github.com/ethereum-optimism/optimism/op-service"

	"github.com/ethereum/go-ethereum/common"
)

const (
	GasLimit                        uint64 = 60_000_000
	BasefeeScalar                   uint32 = 1368
	BlobBaseFeeScalar               uint32 = 801949
	WithdrawalDelaySeconds          uint64 = 302400
	MinProposalSizeBytes            uint64 = 126000
	ChallengePeriodSeconds          uint64 = 86400
	ProofMaturityDelaySeconds       uint64 = 604800
	DisputeGameFinalityDelaySeconds uint64 = 302400
	MIPSVersion                     uint64 = 8
	DisputeGameType                 uint32 = 1 // PERMISSIONED game type
	DisputeMaxGameDepth             uint64 = 73
	DisputeSplitDepth               uint64 = 30
	DisputeClockExtension           uint64 = 10800
	DisputeMaxClockDuration         uint64 = 302400
	Eip1559DenominatorCanyon        uint64 = 250
	Eip1559Denominator              uint64 = 50
	Eip1559Elasticity               uint64 = 6

	ContractsV160Tag        = "op-contracts/v1.6.0"
	ContractsV180Tag        = "op-contracts/v1.8.0-rc.4"
	ContractsV170Beta1L2Tag = "op-contracts/v1.7.0-beta.1+l2-contracts"
	ContractsV200Tag        = "op-contracts/v2.0.0"
	ContractsV300Tag        = "op-contracts/v3.0.0"
	ContractsV400Tag        = "op-contracts/v4.0.0-rc.7"
	CurrentTag              = ContractsV400Tag
)

var DisputeAbsolutePrestate = common.HexToHash("0x038512e02c4c3f7bdaec27d00edf55b7155e0905301e1a88083e4e0a6764d54c")

var VaultMinWithdrawalAmount = mustHexBigFromHex("0x8ac7230489e80000")

var GovernanceTokenOwner = common.HexToAddress("0xDeaDDEaDDeAdDeAdDEAdDEaddeAddEAdDEAdDEad")

func L1VersionsFor(chainID uint64) (validation.Versions, error) {
	switch chainID {
	case 1:
		return validation.StandardVersionsMainnet, nil
	case 11155111:
		return validation.StandardVersionsSepolia, nil
	default:
		return nil, fmt.Errorf("unsupported chain ID: %d", chainID)
	}
}

func GuardianAddressFor(chainID uint64) (common.Address, error) {
	switch chainID {
	case 1:
		return common.Address(validation.StandardConfigRolesMainnet.Guardian), nil
	case 11155111:
		return common.Address(validation.StandardConfigRolesSepolia.Guardian), nil
	default:
		return common.Address{}, fmt.Errorf("unsupported chain ID: %d", chainID)
	}
}

func ChallengerAddressFor(chainID uint64) (common.Address, error) {
	switch chainID {
	case 1:
		return common.Address(validation.StandardConfigRolesMainnet.Challenger), nil
	case 11155111:
		return common.Address(validation.StandardConfigRolesSepolia.Challenger), nil
	default:
		return common.Address{}, fmt.Errorf("unsupported chain ID: %d", chainID)
	}
}

func SuperchainFor(chainID uint64) (superchain.Superchain, error) {
	switch chainID {
	case 1:
		return superchain.GetSuperchain("mainnet")
	case 11155111:
		return superchain.GetSuperchain("sepolia")
	default:
		return superchain.Superchain{}, fmt.Errorf("unsupported chain ID: %d", chainID)
	}
}

func OPCMImplAddressFor(chainID uint64, tag string) (common.Address, error) {
	versionsData, err := L1VersionsFor(chainID)
	if err != nil {
		return common.Address{}, fmt.Errorf("unsupported chainID: %d", chainID)
	}
	versionData, ok := versionsData[validation.Semver(tag)]
	if !ok {
		return common.Address{}, fmt.Errorf("unsupported tag for chainID %d: %s", chainID, tag)
	}
	if versionData.OPContractsManager.Address != nil {
		// op-contracts/v1.8.0 and earlier use proxied opcm
		return common.Address(*versionData.OPContractsManager.Address), nil
	}
	if versionData.OPContractsManager.ImplementationAddress != nil {
		// op-contracts/v2.0.0-rc.1 and later use non-proxied opcm
		return common.Address(*versionData.OPContractsManager.ImplementationAddress), nil
	}
	return common.Address{}, fmt.Errorf("OPContractsManager address is nil for tag %s", tag)
}

// SuperchainProxyAdminAddrFor returns the address of the Superchain ProxyAdmin for the given chain ID.
// These have been verified to be the ProxyAdmin addresses on Mainnet and Sepolia.
// DO NOT MODIFY THIS METHOD WITHOUT CLEARING IT WITH THE EVM SAFETY TEAM.
func SuperchainProxyAdminAddrFor(chainID uint64) (common.Address, error) {
	switch chainID {
	case 1:
		return common.HexToAddress("0x543bA4AADBAb8f9025686Bd03993043599c6fB04"), nil
	case 11155111:
		return common.HexToAddress("0x189aBAAaa82DfC015A588A7dbaD6F13b1D3485Bc"), nil
	default:
		return common.Address{}, fmt.Errorf("unsupported chain ID: %d", chainID)
	}
}

func L1ProxyAdminOwner(chainID uint64) (common.Address, error) {
	switch chainID {
	case 1:
		return common.Address(validation.StandardConfigRolesMainnet.L1ProxyAdminOwner), nil
	case 11155111:
		return common.Address(validation.StandardConfigRolesSepolia.L1ProxyAdminOwner), nil
	default:
		return common.Address{}, fmt.Errorf("unsupported chain ID: %d", chainID)
	}
}

func L2ProxyAdminOwner(chainID uint64) (common.Address, error) {
	switch chainID {
	case 1:
		return common.Address(validation.StandardConfigRolesMainnet.L2ProxyAdminOwner), nil
	case 11155111:
		return common.Address(validation.StandardConfigRolesSepolia.L2ProxyAdminOwner), nil
	default:
		return common.Address{}, fmt.Errorf("unsupported chain ID: %d", chainID)
	}
}

func ProtocolVersionsOwner(chainID uint64) (common.Address, error) {
	switch chainID {
	case 1:
		return common.Address(validation.StandardConfigRolesMainnet.ProtocolVersionsOwner), nil
	case 11155111:
		return common.Address(validation.StandardConfigRolesSepolia.ProtocolVersionsOwner), nil
	default:
		return common.Address{}, fmt.Errorf("unsupported chain ID: %d", chainID)
	}
}

// DefaultHardforkScheduleForTag is used to determine which hardforks should be activated by default given a
// contract tag. For example, passing in v1.6.0 will return all hardforks up to and including Granite. This allows
// OP Deployer to set sane defaults for hardforks. This is not an ideal solution, but it will have to work until we get
// to MCP L2.
func DefaultHardforkScheduleForTag(tag string) *genesis.UpgradeScheduleDeployConfig {
	sched := &genesis.UpgradeScheduleDeployConfig{
		L2GenesisRegolithTimeOffset: op_service.U64UtilPtr(0),
		L2GenesisCanyonTimeOffset:   op_service.U64UtilPtr(0),
		L2GenesisDeltaTimeOffset:    op_service.U64UtilPtr(0),
		L2GenesisEcotoneTimeOffset:  op_service.U64UtilPtr(0),
		L2GenesisFjordTimeOffset:    op_service.U64UtilPtr(0),
		L2GenesisGraniteTimeOffset:  op_service.U64UtilPtr(0),
	}

	switch tag {
	case ContractsV160Tag, ContractsV170Beta1L2Tag:
		return sched
	case ContractsV180Tag, ContractsV200Tag, ContractsV300Tag:
		sched.ActivateForkAtGenesis(rollup.Holocene)
	case ContractsV400Tag:
		sched.ActivateForkAtGenesis(rollup.Holocene)
		sched.ActivateForkAtGenesis(rollup.Isthmus)
	default:
		sched.ActivateForkAtGenesis(rollup.Holocene)
		sched.ActivateForkAtGenesis(rollup.Isthmus)
	}

	return sched
}

func mustHexBigFromHex(hex string) *hexutil.Big {
	num := hexutil.MustDecodeBig(hex)
	hexBig := hexutil.Big(*num)
	return &hexBig
}
