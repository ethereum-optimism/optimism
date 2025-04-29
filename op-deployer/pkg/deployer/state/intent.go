package state

import (
	"errors"
	"fmt"
	"math/big"
	"net/url"
	"reflect"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/artifacts"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/standard"
	"github.com/ethereum-optimism/optimism/op-service/ioutil"
	"github.com/ethereum-optimism/optimism/op-service/jsonutil"
	"github.com/ethereum-optimism/superchain-registry/validation"
)

type IntentType string

const (
	IntentTypeStandard          IntentType = "standard"
	IntentTypeCustom            IntentType = "custom"
	IntentTypeStandardOverrides IntentType = "standard-overrides"
)

var emptyAddress common.Address
var emptyHash common.Hash

type SuperchainProofParams struct {
	WithdrawalDelaySeconds          uint64 `json:"faultGameWithdrawalDelay" toml:"faultGameWithdrawalDelay"`
	MinProposalSizeBytes            uint64 `json:"preimageOracleMinProposalSize" toml:"preimageOracleMinProposalSize"`
	ChallengePeriodSeconds          uint64 `json:"preimageOracleChallengePeriod" toml:"preimageOracleChallengePeriod"`
	ProofMaturityDelaySeconds       uint64 `json:"proofMaturityDelaySeconds" toml:"proofMaturityDelaySeconds"`
	DisputeGameFinalityDelaySeconds uint64 `json:"disputeGameFinalityDelaySeconds" toml:"disputeGameFinalityDelaySeconds"`
	MIPSVersion                     uint64 `json:"mipsVersion" toml:"mipsVersion"`
}

type L1DevGenesisBlockParams struct {
	// Warning: the genesis timestamp will default to time.Now().
	Timestamp uint64 `json:"timestamp"`
	// Gas limit, uses default if 0
	GasLimit uint64 `json:"gasLimit"`
	// Optional. Dencun is always active in L1 dev genesis, so 0 is used as-is if not modified.
	// This may be used to start the chain with high blob fees.
	ExcessBlobGas uint64 `json:"excessBlobGas"`
}

type L1DevGenesisParams struct {
	// BlockParams is the set of genesis-block parameters to use.
	BlockParams L1DevGenesisBlockParams `json:"blockParams" toml:"blockParams"`

	// PragueTimeOffset configures Prague (aka Pectra) to be activated at the given time after L1 dev genesis time.
	PragueTimeOffset *uint64 `json:"pragueTimeOffset" toml:"pragueTimeOffset"`

	// Prefund is a map of addresses to balances (in wei), to prefund in the L1 dev genesis state.
	// This is independent of the "Prefund" functionality that may fund a default 20 test accounts.
	Prefund map[common.Address]*hexutil.U256 `json:"prefund" toml:"prefund"`
}

type Intent struct {
	ConfigType            IntentType         `json:"configType" toml:"configType"`
	L1ChainID             uint64             `json:"l1ChainID" toml:"l1ChainID"`
	SuperchainConfigProxy *common.Address    `json:"superchainConfigProxy" toml:"superchainConfigProxy"`
	SuperchainRoles       *SuperchainRoles   `json:"superchainRoles" toml:"superchainRoles,omitempty"`
	FundDevAccounts       bool               `json:"fundDevAccounts" toml:"fundDevAccounts"`
	UseInterop            bool               `json:"useInterop" toml:"useInterop"`
	L1ContractsLocator    *artifacts.Locator `json:"l1ContractsLocator" toml:"l1ContractsLocator"`
	L2ContractsLocator    *artifacts.Locator `json:"l2ContractsLocator" toml:"l2ContractsLocator"`
	Chains                []*ChainIntent     `json:"chains" toml:"chains"`
	GlobalDeployOverrides map[string]any     `json:"globalDeployOverrides" toml:"globalDeployOverrides"`

	// L1DevGenesisParams is optional. This may be used to customize the L1 genesis when
	// the deployer output is directed to produce a L1 genesis state for development.
	L1DevGenesisParams *L1DevGenesisParams `json:"l1DevGenesisParams"`
}

type SuperchainRoles struct {
	ProxyAdminOwner       common.Address `json:"proxyAdminOwner" toml:"proxyAdminOwner"`
	ProtocolVersionsOwner common.Address `json:"protocolVersionsOwner" toml:"protocolVersionsOwner"`
	Guardian              common.Address `json:"guardian" toml:"guardian"`
}

var ErrSuperchainRoleZeroAddress = errors.New("SuperchainRole is set to zero address")
var ErrL1ContractsLocatorUndefined = errors.New("L1ContractsLocator undefined")
var ErrL2ContractsLocatorUndefined = errors.New("L2ContractsLocator undefined")

func (s *SuperchainRoles) CheckNoZeroAddresses() error {
	val := reflect.ValueOf(*s)
	typ := reflect.TypeOf(*s)

	// Iterate through all the fields
	for i := 0; i < val.NumField(); i++ {
		fieldValue := val.Field(i)
		fieldName := typ.Field(i).Name

		if fieldValue.Interface() == (common.Address{}) {
			return fmt.Errorf("%w: %s", ErrSuperchainRoleZeroAddress, fieldName)
		}
	}
	return nil
}

func (c *Intent) L1ChainIDBig() *big.Int {
	return big.NewInt(int64(c.L1ChainID))
}

func (c *Intent) validateCustomConfig() error {
	if c.L1ContractsLocator == nil ||
		(c.L1ContractsLocator.Tag == "" && c.L1ContractsLocator.URL == &url.URL{}) {
		return ErrL1ContractsLocatorUndefined
	}
	if c.L2ContractsLocator == nil ||
		(c.L2ContractsLocator.Tag == "" && c.L2ContractsLocator.URL == &url.URL{}) {
		return ErrL2ContractsLocatorUndefined
	}

	if c.SuperchainRoles == nil {
		return errors.New("SuperchainRoles is set to nil")
	}
	if err := c.SuperchainRoles.CheckNoZeroAddresses(); err != nil {
		return err
	}

	if len(c.Chains) == 0 {
		return errors.New("must define at least one l2 chain")
	}

	for _, chain := range c.Chains {
		if err := chain.Check(); err != nil {
			return err
		}
	}

	return nil
}

// Ensures the following:
//  1. no zero-values for non-standard fields (user should have populated these)
//  2. no non-standard values for standard fields (user should not have changed these)
func (c *Intent) validateStandardValues() error {
	if err := c.checkL1Prod(); err != nil {
		return err
	}
	if err := c.checkL2Prod(); err != nil {
		return err
	}

	if c.SuperchainConfigProxy != nil {
		return ErrNonStandardValue
	}

	standardSuperchainRoles, err := GetStandardSuperchainRoles(c.L1ChainID)
	if err != nil {
		return fmt.Errorf("error getting standard superchain roles: %w", err)
	}
	if c.SuperchainRoles == nil || *c.SuperchainRoles != *standardSuperchainRoles {
		return fmt.Errorf("SuperchainRoles does not match standard value")
	}

	for _, chain := range c.Chains {
		if err := chain.Check(); err != nil {
			return err
		}
		if chain.Eip1559DenominatorCanyon != standard.Eip1559DenominatorCanyon ||
			chain.Eip1559Denominator != standard.Eip1559Denominator ||
			chain.Eip1559Elasticity != standard.Eip1559Elasticity {
			return fmt.Errorf("%w: chainId=%s", ErrNonStandardValue, chain.ID)
		}
		if len(chain.AdditionalDisputeGames) > 0 {
			return fmt.Errorf("%w: chainId=%s additionalDisputeGames must be nil", ErrNonStandardValue, chain.ID)
		}
	}

	challenger, _ := standard.ChallengerAddressFor(c.L1ChainID)
	l1ProxyAdminOwner, _ := standard.L1ProxyAdminOwner(c.L1ChainID)
	for chainIndex := range c.Chains {
		if c.Chains[chainIndex].Roles.Challenger != challenger {
			return fmt.Errorf("invalid challenger address for chain: %s", c.Chains[chainIndex].ID)
		}
		if c.Chains[chainIndex].Roles.L1ProxyAdminOwner != l1ProxyAdminOwner {
			return fmt.Errorf("invalid l1ProxyAdminOwner address for chain: %s", c.Chains[chainIndex].ID)
		}
	}

	return nil
}

func GetStandardSuperchainRoles(l1ChainId uint64) (*SuperchainRoles, error) {
	proxyAdminOwner, err := standard.L1ProxyAdminOwner(l1ChainId)
	if err != nil {
		return nil, fmt.Errorf("error getting L1ProxyAdminOwner: %w", err)
	}
	guardian, err := standard.GuardianAddressFor(l1ChainId)
	if err != nil {
		return nil, fmt.Errorf("error getting guardian address: %w", err)
	}
	protocolVersionsOwner, err := standard.ProtocolVersionsOwner(l1ChainId)
	if err != nil {
		return nil, fmt.Errorf("error getting protocol versions owner: %w", err)
	}

	superchainRoles := &SuperchainRoles{
		ProxyAdminOwner:       proxyAdminOwner,
		ProtocolVersionsOwner: protocolVersionsOwner,
		Guardian:              guardian,
	}

	return superchainRoles, nil
}

func (c *Intent) Check() error {
	if c.L1ChainID == 0 {
		return fmt.Errorf("l1ChainID cannot be 0")
	}

	if c.L1ContractsLocator == nil {
		return ErrL1ContractsLocatorUndefined
	}

	if c.L2ContractsLocator == nil {
		return ErrL2ContractsLocatorUndefined
	}

	var err error
	switch c.ConfigType {
	case IntentTypeStandard:
		err = c.validateStandardValues()
	case IntentTypeCustom:
		err = c.validateCustomConfig()
	case IntentTypeStandardOverrides:
		err = c.validateCustomConfig()
	default:
		return fmt.Errorf("intent-type unsupported: %s", c.ConfigType)
	}
	if err != nil {
		return fmt.Errorf("failed to validate intent-type=%s: %w", c.ConfigType, err)
	}

	return nil
}

func (c *Intent) Chain(id common.Hash) (*ChainIntent, error) {
	for i := range c.Chains {
		if c.Chains[i].ID == id {
			return c.Chains[i], nil
		}
	}

	return nil, fmt.Errorf("chain %d not found", id)
}

func (c *Intent) WriteToFile(path string) error {
	return jsonutil.WriteTOML(c, ioutil.ToAtomicFile(path, 0o755))
}

func (c *Intent) checkL1Prod() error {
	versions, err := standard.L1VersionsFor(c.L1ChainID)
	if err != nil {
		return err
	}

	if _, ok := versions[validation.Semver(c.L1ContractsLocator.Tag)]; !ok {
		return fmt.Errorf("tag '%s' not found in standard versions", c.L1ContractsLocator.Tag)
	}

	return nil
}

func (c *Intent) checkL2Prod() error {
	_, err := standard.ArtifactsURLForTag(c.L2ContractsLocator.Tag)
	return err
}

func NewIntent(configType IntentType, l1ChainId uint64, l2ChainIds []common.Hash) (intent Intent, err error) {
	switch configType {
	case IntentTypeCustom:
		intent, err = NewIntentCustom(l1ChainId, l2ChainIds)

	case IntentTypeStandard:
		intent, err = NewIntentStandard(l1ChainId, l2ChainIds)

	case IntentTypeStandardOverrides:
		intent, err = NewIntentStandardOverrides(l1ChainId, l2ChainIds)

	default:
		return Intent{}, fmt.Errorf("intent type not supported: %s (valid types: %s, %s, %s)", configType, IntentTypeStandard, IntentTypeCustom, IntentTypeStandardOverrides)
	}
	if err != nil {
		return
	}
	intent.ConfigType = configType
	return
}

// Sets all Intent fields to their zero value with the expectation that the
// user will populate the values before running 'apply'
func NewIntentCustom(l1ChainId uint64, l2ChainIds []common.Hash) (Intent, error) {
	intent := Intent{
		ConfigType:         IntentTypeCustom,
		L1ChainID:          l1ChainId,
		L1ContractsLocator: &artifacts.Locator{URL: &url.URL{}},
		L2ContractsLocator: &artifacts.Locator{URL: &url.URL{}},
		SuperchainRoles:    &SuperchainRoles{},
	}

	for _, l2ChainID := range l2ChainIds {
		intent.Chains = append(intent.Chains, &ChainIntent{
			ID: l2ChainID,
		})
	}
	return intent, nil
}

func NewIntentStandard(l1ChainId uint64, l2ChainIds []common.Hash) (Intent, error) {
	intent := Intent{
		ConfigType:         IntentTypeStandard,
		L1ChainID:          l1ChainId,
		L1ContractsLocator: artifacts.DefaultL1ContractsLocator,
		L2ContractsLocator: artifacts.DefaultL2ContractsLocator,
	}

	superchainRoles, err := GetStandardSuperchainRoles(l1ChainId)
	if err != nil {
		return Intent{}, fmt.Errorf("error getting standard superchain roles: %w", err)
	}
	intent.SuperchainRoles = superchainRoles

	challenger, err := standard.ChallengerAddressFor(l1ChainId)
	if err != nil {
		return Intent{}, fmt.Errorf("error getting challenger address: %w", err)
	}
	l1ProxyAdminOwner, err := standard.L1ProxyAdminOwner(l1ChainId)
	if err != nil {
		return Intent{}, fmt.Errorf("error getting L1ProxyAdminOwner: %w", err)
	}
	l2ProxyAdminOwner, err := standard.L2ProxyAdminOwner(l1ChainId)
	if err != nil {
		return Intent{}, fmt.Errorf("error getting L2ProxyAdminOwner: %w", err)
	}

	for _, l2ChainID := range l2ChainIds {
		intent.Chains = append(intent.Chains, &ChainIntent{
			ID:                       l2ChainID,
			Eip1559DenominatorCanyon: standard.Eip1559DenominatorCanyon,
			Eip1559Denominator:       standard.Eip1559Denominator,
			Eip1559Elasticity:        standard.Eip1559Elasticity,
			Roles: ChainRoles{
				Challenger:        challenger,
				L1ProxyAdminOwner: l1ProxyAdminOwner,
				L2ProxyAdminOwner: l2ProxyAdminOwner,
			},
		})
	}
	return intent, nil
}

func NewIntentStandardOverrides(l1ChainId uint64, l2ChainIds []common.Hash) (Intent, error) {
	intent, err := NewIntentStandard(l1ChainId, l2ChainIds)
	if err != nil {
		return Intent{}, err
	}
	intent.ConfigType = IntentTypeStandardOverrides

	return intent, nil
}
