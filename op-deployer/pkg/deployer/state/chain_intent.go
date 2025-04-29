package state

import (
	"fmt"
	"reflect"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
)

type VMType string

const (
	VMTypeAlphabet = "ALPHABET"
	VMTypeCannon1  = "CANNON1" // Legacy: corresponds to 32-bit MIPS VM
	VMTypeCannon2  = "CANNON2" // Legacy: corresponds to 64-bit MIPS VM StateVersion 6
	VMTypeCannon6  = "CANNON6" // Corresponds to VM State Version 6: https://github.com/ethereum-optimism/optimism/blob/4c05241bc534ae5837007c32995fc62f3dd059b6/cannon/mipsevm/versions/version.go#L25-L25
	VMTypeCannon7  = "CANNON7" // Corresponds to VM State Version 7: https://github.com/ethereum-optimism/optimism/blob/4c05241bc534ae5837007c32995fc62f3dd059b6/cannon/mipsevm/versions/version.go#L27-L27
)

func (v VMType) MipsVersion() uint64 {
	switch v {
	case VMTypeCannon1:
		return 1
	case VMTypeCannon2, VMTypeCannon6:
		return 6
	case VMTypeCannon7:
		return 7
	default:
		// Not a mips VM - return empty value
		return 0
	}
}

type ChainProofParams struct {
	DisputeGameType                         uint32      `json:"respectedGameType" toml:"respectedGameType"`
	DisputeAbsolutePrestate                 common.Hash `json:"faultGameAbsolutePrestate" toml:"faultGameAbsolutePrestate"`
	DisputeMaxGameDepth                     uint64      `json:"faultGameMaxDepth" toml:"faultGameMaxDepth"`
	DisputeSplitDepth                       uint64      `json:"faultGameSplitDepth" toml:"faultGameSplitDepth"`
	DisputeClockExtension                   uint64      `json:"faultGameClockExtension" toml:"faultGameClockExtension"`
	DisputeMaxClockDuration                 uint64      `json:"faultGameMaxClockDuration" toml:"faultGameMaxClockDuration"`
	DangerouslyAllowCustomDisputeParameters bool        `json:"dangerouslyAllowCustomDisputeParameters" toml:"dangerouslyAllowCustomDisputeParameters"`
}

type AdditionalDisputeGame struct {
	ChainProofParams
	VMType                       VMType
	UseCustomOracle              bool
	OracleMinProposalSize        uint64
	OracleChallengePeriodSeconds uint64
	MakeRespected                bool
}

type L2DevGenesisParams struct {
	// Prefund is a map of addresses to balances (in wei), to prefund in the L2 dev genesis state.
	// This is independent of the "Prefund" functionality that may fund a default 20 test accounts.
	Prefund map[common.Address]*hexutil.U256 `json:"prefund" toml:"prefund"`
}

type ChainIntent struct {
	ID                         common.Hash               `json:"id" toml:"id"`
	BaseFeeVaultRecipient      common.Address            `json:"baseFeeVaultRecipient" toml:"baseFeeVaultRecipient"`
	L1FeeVaultRecipient        common.Address            `json:"l1FeeVaultRecipient" toml:"l1FeeVaultRecipient"`
	SequencerFeeVaultRecipient common.Address            `json:"sequencerFeeVaultRecipient" toml:"sequencerFeeVaultRecipient"`
	Eip1559DenominatorCanyon   uint64                    `json:"eip1559DenominatorCanyon" toml:"eip1559DenominatorCanyon"`
	Eip1559Denominator         uint64                    `json:"eip1559Denominator" toml:"eip1559Denominator"`
	Eip1559Elasticity          uint64                    `json:"eip1559Elasticity" toml:"eip1559Elasticity"`
	Roles                      ChainRoles                `json:"roles" toml:"roles"`
	DeployOverrides            map[string]any            `json:"deployOverrides" toml:"deployOverrides"`
	DangerousAltDAConfig       genesis.AltDADeployConfig `json:"dangerousAltDAConfig,omitempty" toml:"dangerousAltDAConfig,omitempty"`
	AdditionalDisputeGames     []AdditionalDisputeGame   `json:"dangerousAdditionalDisputeGames" toml:"dangerousAdditionalDisputeGames,omitempty"`
	OperatorFeeScalar          uint32                    `json:"operatorFeeScalar,omitempty" toml:"operatorFeeScalar,omitempty"`
	OperatorFeeConstant        uint64                    `json:"operatorFeeConstant,omitempty" toml:"operatorFeeConstant,omitempty"`
	L1StartBlockHash           *common.Hash              `json:"l1StartBlockHash,omitempty" toml:"l1StartBlockHash,omitempty"`

	// Optional. For development purposes only. Only enabled if the operation mode targets a genesis-file output.
	L2DevGenesisParams *L2DevGenesisParams `json:"l2DevGenesisParams,omitempty" toml:"l2DevGenesisParams,omitempty"`
}

type ChainRoles struct {
	L1ProxyAdminOwner common.Address `json:"l1ProxyAdminOwner" toml:"l1ProxyAdminOwner"`
	L2ProxyAdminOwner common.Address `json:"l2ProxyAdminOwner" toml:"l2ProxyAdminOwner"`
	SystemConfigOwner common.Address `json:"systemConfigOwner" toml:"systemConfigOwner"`
	UnsafeBlockSigner common.Address `json:"unsafeBlockSigner" toml:"unsafeBlockSigner"`
	Batcher           common.Address `json:"batcher" toml:"batcher"`
	Proposer          common.Address `json:"proposer" toml:"proposer"`
	Challenger        common.Address `json:"challenger" toml:"challenger"`
}

var ErrChainRoleZeroAddress = fmt.Errorf("ChainRole is set to zero address")
var ErrFeeVaultZeroAddress = fmt.Errorf("chain has a fee vault set to zero address")
var ErrNonStandardValue = fmt.Errorf("chain contains non-standard config value")
var ErrEip1559ZeroValue = fmt.Errorf("eip1559 param is set to zero value")

func (c *ChainIntent) Check() error {
	if c.ID == emptyHash {
		return fmt.Errorf("id must be set")
	}

	if err := c.Roles.CheckNoZeroAddresses(); err != nil {
		return err
	}

	if c.Eip1559DenominatorCanyon == 0 ||
		c.Eip1559Denominator == 0 ||
		c.Eip1559Elasticity == 0 {
		return fmt.Errorf("%w: chainId=%s", ErrEip1559ZeroValue, c.ID)
	}
	if c.BaseFeeVaultRecipient == emptyAddress ||
		c.L1FeeVaultRecipient == emptyAddress ||
		c.SequencerFeeVaultRecipient == emptyAddress {
		return fmt.Errorf("%w: chainId=%s", ErrFeeVaultZeroAddress, c.ID)
	}

	if c.DangerousAltDAConfig.UseAltDA {
		return c.DangerousAltDAConfig.Check(nil)
	}

	return nil
}

// Returns an error if any fields in ChainRoles is set to common.Address{}
func (cr *ChainRoles) CheckNoZeroAddresses() error {
	val := reflect.ValueOf(*cr)
	typ := reflect.TypeOf(*cr)

	// Iterate through all the fields
	for i := 0; i < val.NumField(); i++ {
		fieldValue := val.Field(i)
		fieldName := typ.Field(i).Name

		if fieldValue.Interface() == (common.Address{}) {
			return fmt.Errorf("%w: %s", ErrChainRoleZeroAddress, fieldName)
		}
	}

	return nil
}
