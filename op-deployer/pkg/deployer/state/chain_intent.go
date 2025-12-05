package state

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm/versions"
	"github.com/ethereum-optimism/optimism/op-chain-ops/addresses"
	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
)

type VMType string

const (
	VMTypeAlphabet   = "ALPHABET"
	VMTypeCannon     = "CANNON"      // Corresponds to the currently released Cannon StateVersion. See: https://github.com/ethereum-optimism/optimism/blob/4c05241bc534ae5837007c32995fc62f3dd059b6/cannon/mipsevm/versions/version.go
	VMTypeCannonNext = "CANNON-NEXT" // Corresponds to the next in-development Cannon StateVersion. See: https://github.com/ethereum-optimism/optimism/blob/4c05241bc534ae5837007c32995fc62f3dd059b6/cannon/mipsevm/versions/version.go
)

func (v VMType) MipsVersion() uint64 {
	switch v {
	case VMTypeCannon:
		return uint64(versions.GetCurrentVersion())
	case VMTypeCannonNext:
		return uint64(versions.GetExperimentalVersion())
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
	VMType        VMType
	MakeRespected bool
}

type L2DevGenesisParams struct {
	// Prefund is a map of addresses to balances (in wei), to prefund in the L2 dev genesis state.
	// This is independent of the "Prefund" functionality that may fund a default 20 test accounts.
	Prefund map[common.Address]*hexutil.U256 `json:"prefund" toml:"prefund"`
}

type CustomGasToken struct {
	Name                     string         `json:"name,omitempty" toml:"name,omitempty"`
	Symbol                   string         `json:"symbol,omitempty" toml:"symbol,omitempty"`
	InitialLiquidity         *hexutil.Big   `json:"initialLiquidity" toml:"initialLiquidity"`
	LiquidityControllerOwner common.Address `json:"liquidityControllerOwner" toml:"liquidityControllerOwner"`
}

type ChainIntent struct {
	ID                         common.Hash               `json:"id" toml:"id"`
	BaseFeeVaultRecipient      common.Address            `json:"baseFeeVaultRecipient" toml:"baseFeeVaultRecipient"`
	L1FeeVaultRecipient        common.Address            `json:"l1FeeVaultRecipient" toml:"l1FeeVaultRecipient"`
	SequencerFeeVaultRecipient common.Address            `json:"sequencerFeeVaultRecipient" toml:"sequencerFeeVaultRecipient"`
	OperatorFeeVaultRecipient  common.Address            `json:"operatorFeeVaultRecipient" toml:"operatorFeeVaultRecipient"`
	Eip1559DenominatorCanyon   uint64                    `json:"eip1559DenominatorCanyon" toml:"eip1559DenominatorCanyon"`
	Eip1559Denominator         uint64                    `json:"eip1559Denominator" toml:"eip1559Denominator"`
	Eip1559Elasticity          uint64                    `json:"eip1559Elasticity" toml:"eip1559Elasticity"`
	GasLimit                   uint64                    `json:"gasLimit" toml:"gasLimit"`
	Roles                      ChainRoles                `json:"roles" toml:"roles"`
	DeployOverrides            map[string]any            `json:"deployOverrides" toml:"deployOverrides"`
	DangerousAltDAConfig       genesis.AltDADeployConfig `json:"dangerousAltDAConfig,omitempty" toml:"dangerousAltDAConfig,omitempty"`
	AdditionalDisputeGames     []AdditionalDisputeGame   `json:"dangerousAdditionalDisputeGames" toml:"dangerousAdditionalDisputeGames,omitempty"`
	OperatorFeeScalar          uint32                    `json:"operatorFeeScalar,omitempty" toml:"operatorFeeScalar,omitempty"`
	OperatorFeeConstant        uint64                    `json:"operatorFeeConstant,omitempty" toml:"operatorFeeConstant,omitempty"`
	L1StartBlockHash           *common.Hash              `json:"l1StartBlockHash,omitempty" toml:"l1StartBlockHash,omitempty"`
	UseRevenueShare            bool                      `json:"useRevenueShare,omitempty" toml:"useRevenueShare,omitempty"`
	ChainFeesRecipient         common.Address            `json:"chainFeesRecipient,omitempty" toml:"chainFeesRecipient,omitempty"`
	MinBaseFee                 uint64                    `json:"minBaseFee,omitempty" toml:"minBaseFee,omitempty"`
	DAFootprintGasScalar       uint16                    `json:"daFootprintGasScalar,omitempty" toml:"daFootprintGasScalar,omitempty"`
	CustomGasToken             CustomGasToken            `json:"customGasToken" toml:"customGasToken"`

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

var ErrFeeVaultZeroAddress = fmt.Errorf("chain has a fee vault set to zero address")
var ErrGasLimitZeroValue = fmt.Errorf("chain has a gas limit set to zero value")
var ErrNonStandardValue = fmt.Errorf("chain contains non-standard config value")
var ErrEip1559ZeroValue = fmt.Errorf("eip1559 param is set to zero value")
var ErrIncompatibleValue = fmt.Errorf("chain contains incompatible config value")
var ErrRevenueShareZeroAddress = fmt.Errorf("chain has enabled revenue share but recipient is set to zero address")

func (c *ChainIntent) Check() error {
	if c.ID == emptyHash {
		return fmt.Errorf("id must be set")
	}

	if err := addresses.CheckNoZeroAddresses(c.Roles); err != nil {
		return err
	}

	if c.Eip1559DenominatorCanyon == 0 ||
		c.Eip1559Denominator == 0 ||
		c.Eip1559Elasticity == 0 {
		return fmt.Errorf("%w: chainId=%s", ErrEip1559ZeroValue, c.ID)
	}

	if c.GasLimit == 0 {
		return fmt.Errorf("%w: chainId=%s", ErrGasLimitZeroValue, c.ID)
	}

	if c.BaseFeeVaultRecipient == emptyAddress ||
		c.L1FeeVaultRecipient == emptyAddress ||
		c.SequencerFeeVaultRecipient == emptyAddress ||
		c.OperatorFeeVaultRecipient == emptyAddress {
		return fmt.Errorf("%w: chainId=%s", ErrFeeVaultZeroAddress, c.ID)
	}

	// Validate CustomGasToken: if any field is set, both Name and Symbol must be present
	hasName := c.CustomGasToken.Name != ""
	hasSymbol := c.CustomGasToken.Symbol != ""
	hasAnyCustomGasTokenField := hasName || hasSymbol || c.CustomGasToken.InitialLiquidity != nil || c.CustomGasToken.LiquidityControllerOwner != (common.Address{})

	if hasAnyCustomGasTokenField {
		if !hasName {
			return fmt.Errorf("%w: CustomGasToken.Name must be set when using custom gas token, chainId=%s", ErrIncompatibleValue, c.ID)
		}
		if !hasSymbol {
			return fmt.Errorf("%w: CustomGasToken.Symbol must be set when using custom gas token, chainId=%s", ErrIncompatibleValue, c.ID)
		}

		// InitialLiquidity is optional - if not set, type(uint248).max will be used as default
		// But if it IS set, it must be non-negative
		if c.CustomGasToken.InitialLiquidity != nil && c.CustomGasToken.InitialLiquidity.ToInt().Sign() < 0 {
			return fmt.Errorf("%w: CustomGasToken.InitialLiquidity must be non-negative when custom gas token is enabled, chainId=%s", ErrIncompatibleValue, c.ID)
		}
		// LiquidityControllerOwner is optional - if not set, L2ProxyAdminOwner will be used as default
	}

	if c.DangerousAltDAConfig.UseAltDA {
		return c.DangerousAltDAConfig.Check(nil)
	}

	if c.UseRevenueShare {
		if c.ChainFeesRecipient == emptyAddress {
			return fmt.Errorf("%w: chainId=%s", ErrRevenueShareZeroAddress, c.ID)
		}
	}

	return nil
}

// GetInitialLiquidity returns the native asset liquidity amount for the chain.
// If not set and custom gas token is enabled, returns type(uint248).max as the default.
// Otherwise returns zero.
func (c *ChainIntent) GetInitialLiquidity() *big.Int {
	if c.CustomGasToken.InitialLiquidity != nil {
		return c.CustomGasToken.InitialLiquidity.ToInt()
	}

	// If custom gas token is enabled but no liquidity specified, use type(uint248).max
	// This is the safe default: large enough to never go to 0, small enough to never overflow
	if c.IsCustomGasTokenEnabled() {
		maxUint248 := new(big.Int)
		maxUint248.SetString("00ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", 16)
		return maxUint248
	}

	// When CGT is not enabled, return zero to indicate no liquidity
	return big.NewInt(0)
}

// GetLiquidityControllerOwner returns the owner of the LiquidityController.
// If not set in CustomGasToken config, defaults to L2ProxyAdminOwner.
func (c *ChainIntent) GetLiquidityControllerOwner() common.Address {
	if c.CustomGasToken.LiquidityControllerOwner != (common.Address{}) {
		return c.CustomGasToken.LiquidityControllerOwner
	}
	return c.Roles.L2ProxyAdminOwner
}

// IsCustomGasTokenEnabled returns true if custom gas token is enabled.
// It's enabled when both Name and Symbol are provided.
func (c *ChainIntent) IsCustomGasTokenEnabled() bool {
	return c.CustomGasToken.Name != "" && c.CustomGasToken.Symbol != ""
}
