package standard

import (
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum-optimism/optimism/op-core/forks"
	opparams "github.com/ethereum-optimism/optimism/op-core/params"
	"github.com/ethereum-optimism/optimism/op-core/superchain"

	"github.com/ethereum-optimism/superchain-registry/validation"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
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
	// DisputeGameType is the SUPER_PERMISSIONED game type. DeployOPChain requires the initial game
	// type to match the OPCM's family, and SUPER_ROOT_GAMES_MIGRATION is enabled by default, so the
	// permissioned selector for a standard deploy is the super root one.
	// TODO(#21662): revisit with the broader SuperRootGamesMigration cleanup.
	DisputeGameType          uint32 = 5
	DisputeMaxGameDepth      uint64 = 73
	DisputeSplitDepth        uint64 = 30
	DisputeClockExtension    uint64 = 10800
	DisputeMaxClockDuration  uint64 = 302400
	Eip1559DenominatorCanyon uint64 = 250
	Eip1559Denominator       uint64 = 50
	Eip1559Elasticity        uint64 = 6

	// TODO(#20916): This value should be replaced with a benchmark based on the time it takes to perform a full
	// L2 genesis deployment.
	// DefaultGenesisTimeOffsetSeconds is the default offset added to the L1 anchor block's
	// timestamp to produce the committed L2 genesis timestamp.
	DefaultGenesisTimeOffsetSeconds uint64 = 21600 // 6 hours

	ContractsV160Tag        = "op-contracts/v1.6.0"
	ContractsV180Tag        = "op-contracts/v1.8.0-rc.4"
	ContractsV170Beta1L2Tag = "op-contracts/v1.7.0-beta.1+l2-contracts"
	ContractsV200Tag        = "op-contracts/v2.0.0"
	ContractsV300Tag        = "op-contracts/v3.0.0"
	ContractsV400Tag        = "op-contracts/v4.0.0-rc.7"
	ContractsV410Tag        = "op-contracts/v4.1.0"
	ContractsV500Tag        = "op-contracts/v5.0.0"
	ContractsV600Tag        = "op-contracts/v6.0.0-rc.2"
	ContractsV700Tag        = "op-contracts/v7.0.0-rc.4"
	ContractsV800PCDTag     = "op-contracts/v8.0.0-pcdtest"
	CurrentTag              = ContractsV800PCDTag
)

var DisputeAbsolutePrestate = common.HexToHash("0x038512e02c4c3f7bdaec27d00edf55b7155e0905301e1a88083e4e0a6764d54c")

var VaultMinWithdrawalAmount = mustHexBigFromHex("0x8ac7230489e80000")

var GovernanceTokenOwner = common.HexToAddress("0xDeaDDEaDDeAdDeAdDEAdDEaddeAddEAdDEAdDEad")

// PlaceholderAddress is a non-zero sentinel address. It's used  as the default deployer when no private key is provided.
// The same-sender check keys off this value to skip when no real deployer is set.
var PlaceholderAddress = common.Address{0x01}

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

// DefaultHardforkSchedule is used to determine which hardforks should be activated by default.
// It activates, at genesis, the most recent fork that has an activation timestamp scheduled on
// OP Mainnet.
func DefaultHardforkSchedule() *genesis.UpgradeScheduleDeployConfig {
	sched := &genesis.UpgradeScheduleDeployConfig{}
	sched.ActivateForkAtGenesis(defaultHardfork())

	return sched
}

var defaultHardfork = sync.OnceValue(func() forks.Name {
	chain, err := superchain.GetChain(opparams.OPMainnetChainID)
	if err != nil {
		panic(fmt.Errorf("get op mainnet chain: %w", err))
	}
	chainConfig, err := chain.Config()
	if err != nil {
		panic(fmt.Errorf("load op mainnet chain config: %w", err))
	}
	return latestScheduledMainlineFork(chainConfig.Hardforks)
})

func latestScheduledMainlineFork(config superchain.HardforkConfig) forks.Name {
	configValue := reflect.ValueOf(config)
	mainlineForks := forks.From(forks.Canyon)
	for i := len(mainlineForks) - 1; i >= 0; i-- {
		fork := mainlineForks[i]
		// HardforkConfig fields follow the <ForkName>Time convention. Looking them up
		// from forks.All keeps this selection current when a new mainline fork is added.
		field := configValue.FieldByNameFunc(func(name string) bool {
			return strings.EqualFold(strings.TrimSuffix(name, "Time"), string(fork))
		})
		if !field.IsValid() {
			panic(fmt.Sprintf("mainline fork %q is missing from superchain.HardforkConfig", fork))
		}
		activationTime, ok := field.Interface().(*uint64)
		if !ok {
			panic(fmt.Sprintf("superchain.HardforkConfig field for %q must be *uint64", fork))
		}
		if activationTime != nil {
			return fork
		}
	}

	// Regolith is active at genesis for every registry-backed rollup config, but its
	// activation is implicit and is therefore not represented in HardforkConfig.
	return forks.Regolith
}

func mustHexBigFromHex(hex string) *hexutil.Big {
	num := hexutil.MustDecodeBig(hex)
	hexBig := hexutil.Big(*num)
	return &hexBig
}
