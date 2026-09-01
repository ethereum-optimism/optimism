package state

import (
	"fmt"
	"math"
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
	VMTypeZK         = "ZK"          // ZK dispute game — uses a ZK verifier instead of a MIPS VM, with super-root semantics.
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

// FaultGameAbsolutePrestateOverrideKey names the absolute prestate override.
const FaultGameAbsolutePrestateOverrideKey = "faultGameAbsolutePrestate"

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
	// ZKDisputeGame holds ZK-specific configuration. Only used when VMType == VMTypeZK.
	ZKDisputeGame *ZKDisputeGameParams `json:"zkDisputeGame,omitempty" toml:"zkDisputeGame,omitempty"`
}

// ZKDisputeGameParams holds the configuration for a ZK dispute game in the upgrade pipeline.
type ZKDisputeGameParams struct {
	AbsolutePrestate     common.Hash  `json:"absolutePrestate" toml:"absolutePrestate"`
	MaxChallengeDuration uint64       `json:"maxChallengeDuration" toml:"maxChallengeDuration"`
	MaxProveDuration     uint64       `json:"maxProveDuration" toml:"maxProveDuration"`
	ChallengerBond       *hexutil.Big `json:"challengerBond" toml:"challengerBond"`
}

type L2DevGenesisParams struct {
	// Prefund is a map of addresses to balances (in wei), to prefund in the L2 dev genesis state.
	// This is independent of the "Prefund" functionality that may fund a default 20 test accounts.
	Prefund map[common.Address]*hexutil.U256 `json:"prefund" toml:"prefund"`
}

// PrivateInterop configures a private chain. The public projection is derived deterministically
// from this chain's genesis and is not a second op-deployer role.
type PrivateInterop struct {
	// CounterpartyChainID is the chain holding the ETHLockVault the private chain's
	// NativeMintBridge mints against.
	CounterpartyChainID uint64 `json:"counterpartyChainID,omitempty" toml:"counterpartyChainID,omitempty"`

	// LockVault is the ETHLockVault's address on the counterparty chain.
	LockVault common.Address `json:"lockVault,omitempty" toml:"lockVault,omitempty"`

	// ExtraEmitters are additional application contracts whose logs are published by the public
	// projection, beyond the two standard interop predeploys. The generated rollup config is the
	// runtime source of truth for this set.
	ExtraEmitters []common.Address `json:"extraEmitters,omitempty" toml:"extraEmitters,omitempty"`
}

// Check validates the private interop configuration.
func (p *PrivateInterop) Check() error {
	if p == nil {
		return nil
	}
	if p.CounterpartyChainID == 0 {
		return fmt.Errorf("%w: privateInterop.counterpartyChainID must be set", ErrIncompatibleValue)
	}
	if p.LockVault == (common.Address{}) {
		return fmt.Errorf("%w: privateInterop.lockVault must be set", ErrIncompatibleValue)
	}
	return nil
}

// ResourceConfig overrides the L1 resource config a chain's SystemConfig is initialized with.
// Absent (nil) on every ordinary chain, which then gets the gas-limit-derived default.
//
// The reason to set one at all: a chain becomes undepositable, on a completely stock portal, when
// it is initialized with MaxResourceLimit = 0. Every OptimismPortal.depositTransaction is metered
// against that limit with a gas limit of at least 21000, so every deposit reverts on L1 and no user
// deposit ever exists for derivation or attribute handling to see. That is what a private interop
// chain needs: its ETH solvency story rests on the public projection never minting ETH out of a
// deposit.
//
// It can only be set at initialization -- this SystemConfig has no owner-callable setter -- so a
// live owner can still rotate the batcher key without being able to reopen deposits, and undoing it
// is a governed upgrade-and-reinitialize like any other L1 change.
type ResourceConfig struct {
	// MaxResourceLimit is the deposit gas budget per L1 block. Zero closes deposits.
	MaxResourceLimit uint32 `json:"maxResourceLimit" toml:"maxResourceLimit"`
	// ElasticityMultiplier must be positive and must divide MaxResourceLimit exactly. Use 1
	// alongside a zero limit.
	ElasticityMultiplier uint8 `json:"elasticityMultiplier" toml:"elasticityMultiplier"`
	// SystemTxMaxGas is the gas reserved for the L1 attributes deposit.
	SystemTxMaxGas uint32 `json:"systemTxMaxGas" toml:"systemTxMaxGas"`
}

const PrivateInteropSystemTxMaxGas uint32 = 1_000_000

// ClosedDepositsResourceConfig is the resource configuration every private-interop chain derives.
// It is deliberately not a deployment option: selecting the chain type closes deposits
// structurally, so omitting an otherwise-independent intent field cannot reopen them.
func ClosedDepositsResourceConfig() *ResourceConfig {
	return &ResourceConfig{
		MaxResourceLimit:     0,
		ElasticityMultiplier: 1,
		SystemTxMaxGas:       PrivateInteropSystemTxMaxGas,
	}
}

// EffectiveResourceConfig returns the configuration the deployer must apply. A private-interop
// stanza takes precedence over the ordinary optional override because closed deposits are part of
// that chain type, not an operator-selected tuning value.
func (c *ChainIntent) EffectiveResourceConfig() *ResourceConfig {
	if c.PrivateInterop != nil {
		return ClosedDepositsResourceConfig()
	}
	return c.ResourceConfig
}

// Check validates the override against the constraints SystemConfig._setResourceConfig enforces,
// so a bad value is rejected while the intent is still a file rather than halfway through a deploy.
func (r *ResourceConfig) Check(gasLimit uint64) error {
	if r == nil {
		return nil
	}
	if r.ElasticityMultiplier == 0 {
		return fmt.Errorf("%w: resourceConfig.elasticityMultiplier must be positive", ErrIncompatibleValue)
	}
	if uint64(r.MaxResourceLimit)%uint64(r.ElasticityMultiplier) != 0 {
		return fmt.Errorf("%w: resourceConfig.maxResourceLimit must be a multiple of elasticityMultiplier",
			ErrIncompatibleValue)
	}
	if uint64(r.MaxResourceLimit)+uint64(r.SystemTxMaxGas) > gasLimit {
		return fmt.Errorf("%w: resourceConfig.maxResourceLimit + systemTxMaxGas exceeds the chain's gas limit",
			ErrIncompatibleValue)
	}
	return nil
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
	MinBaseFee                 uint64                    `json:"minBaseFee,omitempty" toml:"minBaseFee,omitempty"`
	DAFootprintGasScalar       uint16                    `json:"daFootprintGasScalar,omitempty" toml:"daFootprintGasScalar,omitempty"`
	CustomGasToken             CustomGasToken            `json:"customGasToken" toml:"customGasToken"`
	PrivateInterop             *PrivateInterop           `json:"privateInterop,omitempty" toml:"privateInterop,omitempty"`
	ResourceConfig             *ResourceConfig           `json:"resourceConfig,omitempty" toml:"resourceConfig,omitempty"`

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
var ErrZKDisputeGameMissingParams = fmt.Errorf("ZK dispute game is missing required params")
var ErrZKDisputeGameParamOutOfRange = fmt.Errorf("ZK dispute game param is out of range")

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

	if c.PrivateInterop != nil && c.ResourceConfig != nil {
		closed := ClosedDepositsResourceConfig()
		if *c.ResourceConfig != *closed {
			return fmt.Errorf("%w: private interop derives a closed-deposit resource config; remove the conflicting resourceConfig override, chainId=%s",
				ErrIncompatibleValue, c.ID)
		}
	}

	if err := c.EffectiveResourceConfig().Check(c.GasLimit); err != nil {
		return fmt.Errorf("%w: chainId=%s", err, c.ID)
	}

	if err := c.PrivateInterop.Check(); err != nil {
		return fmt.Errorf("%w: chainId=%s", err, c.ID)
	}

	if c.PrivateInterop != nil && !c.IsCustomGasTokenEnabled() {
		return fmt.Errorf("%w: a private interop chain requires a custom gas token, chainId=%s", ErrIncompatibleValue, c.ID)
	}
	if c.DangerousAltDAConfig.UseAltDA {
		return c.DangerousAltDAConfig.Check(nil)
	}

	for _, game := range c.AdditionalDisputeGames {
		if game.VMType == VMTypeZK {
			if game.ZKDisputeGame == nil {
				return fmt.Errorf("%w: zkDisputeGame config must be set when VMType is ZK, chainId=%s", ErrZKDisputeGameMissingParams, c.ID)
			}
			if game.ZKDisputeGame.AbsolutePrestate == (common.Hash{}) {
				return fmt.Errorf("%w: AbsolutePrestate must not be zero, chainId=%s", ErrZKDisputeGameMissingParams, c.ID)
			}
			if game.ZKDisputeGame.MaxChallengeDuration == 0 {
				return fmt.Errorf("%w: MaxChallengeDuration must be > 0, chainId=%s", ErrZKDisputeGameMissingParams, c.ID)
			}
			// Capped at uint32 max so block.timestamp + duration cannot overflow the game's uint64
			// deadline cast, which would place the deadline in the past.
			if game.ZKDisputeGame.MaxChallengeDuration > math.MaxUint32 {
				return fmt.Errorf("%w: MaxChallengeDuration must be <= %d, chainId=%s", ErrZKDisputeGameParamOutOfRange, uint64(math.MaxUint32), c.ID)
			}
			if game.ZKDisputeGame.MaxProveDuration == 0 {
				return fmt.Errorf("%w: MaxProveDuration must be > 0, chainId=%s", ErrZKDisputeGameMissingParams, c.ID)
			}
			if game.ZKDisputeGame.MaxProveDuration > math.MaxUint32 {
				return fmt.Errorf("%w: MaxProveDuration must be <= %d, chainId=%s", ErrZKDisputeGameParamOutOfRange, uint64(math.MaxUint32), c.ID)
			}
			if game.ZKDisputeGame.ChallengerBond == nil || game.ZKDisputeGame.ChallengerBond.ToInt().Sign() <= 0 {
				return fmt.Errorf("%w: ChallengerBond must be set to a positive value, chainId=%s", ErrZKDisputeGameMissingParams, c.ID)
			}
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
