package intentbuilder

import (
	"fmt"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/ethereum-optimism/optimism/op-chain-ops/addresses"
	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/artifacts"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/standard"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type SuperchainID string

type L1Configurator interface {
	WithChainID(chainID eth.ChainID) L1Configurator
	WithTimestamp(v uint64) L1Configurator
	WithGasLimit(v uint64) L1Configurator
	WithExcessBlobGas(v uint64) L1Configurator
	WithPragueOffset(v uint64) L1Configurator
	WithPrefundedAccount(addr common.Address, amount uint256.Int) L1Configurator
}

type SuperchainConfigurator interface {
	ID() SuperchainID
	L1ChainID() eth.ChainID
	WithSuperchainConfigProxy(address common.Address) SuperchainConfigurator
	WithProxyAdminOwner(address common.Address) SuperchainConfigurator
	WithGuardian(address common.Address) SuperchainConfigurator
	WithProtocolVersionsOwner(address common.Address) SuperchainConfigurator
	WithChallenger(address common.Address) SuperchainConfigurator
}

type L2Configurator interface {
	L1Config() L1Configurator
	ChainID() eth.ChainID
	WithBlockTime(uint64)
	WithL1StartBlockHash(hash common.Hash)
	WithAdditionalDisputeGames(games []state.AdditionalDisputeGame)
	WithFinalizationPeriodSeconds(value uint64)
	ContractsConfigurator
	L2VaultsConfigurator
	L2RolesConfigurator
	L2FeesConfigurator
	L2HardforkConfigurator
	WithPrefundedAccount(addr common.Address, amount uint256.Int) L2Configurator
}

type ContractsConfigurator interface {
	WithL1ContractsLocator(url string)
	WithL2ContractsLocator(url string)
}

type L2VaultsConfigurator interface {
	WithBaseFeeVaultRecipient(address common.Address)
	WithSequencerFeeVaultRecipient(address common.Address)
	WithL1FeeVaultRecipient(address common.Address)
}

type L2RolesConfigurator interface {
	WithL1ProxyAdminOwner(address common.Address)
	WithL2ProxyAdminOwner(address common.Address)
	WithSystemConfigOwner(address common.Address)
	WithUnsafeBlockSigner(address common.Address)
	WithBatcher(address common.Address)
	WithProposer(address common.Address)
	WithChallenger(address common.Address)
}

type L2FeesConfigurator interface {
	WithEIP1559DenominatorCanyon(uint64)
	WithEIP1559Denominator(uint64)
	WithEIP1559Elasticity(uint64)
	WithOperatorFeeScalar(uint64)
	WithOperatorFeeConstant(uint64)
}

type L2HardforkConfigurator interface {
	WithForkAtGenesis(fork rollup.ForkName)
	WithForkAtOffset(fork rollup.ForkName, offset *uint64)
}

type Builder interface {
	WithL1ContractsLocator(loc *artifacts.Locator) Builder
	WithL2ContractsLocator(loc *artifacts.Locator) Builder

	WithSuperchain() (Builder, SuperchainConfigurator)
	WithL1(l1ChainID eth.ChainID) (Builder, L1Configurator)
	WithL2(l2ChainID eth.ChainID) (Builder, L2Configurator)
	L2s() (out []L2Configurator)
	Build() (*state.Intent, error)

	WithGlobalOverride(key string, value any) Builder
}

func WithDevkeyVaults(t require.TestingT, dk devkeys.Keys, configurator L2Configurator) {
	addrFor := RoleToAddrProvider(t, dk, configurator.ChainID())
	configurator.WithBaseFeeVaultRecipient(addrFor(devkeys.BaseFeeVaultRecipientRole))
	configurator.WithSequencerFeeVaultRecipient(addrFor(devkeys.SequencerFeeVaultRecipientRole))
	configurator.WithL1FeeVaultRecipient(addrFor(devkeys.L1FeeVaultRecipientRole))
}

func WithDevkeyL2Roles(t require.TestingT, dk devkeys.Keys, configurator L2Configurator) {
	addrFor := RoleToAddrProvider(t, dk, configurator.ChainID())
	configurator.WithL2ProxyAdminOwner(addrFor(devkeys.L2ProxyAdminOwnerRole))
	configurator.WithSystemConfigOwner(addrFor(devkeys.SystemConfigOwner))
	configurator.WithUnsafeBlockSigner(addrFor(devkeys.SequencerP2PRole))
	configurator.WithBatcher(addrFor(devkeys.BatcherRole))
	configurator.WithProposer(addrFor(devkeys.ProposerRole))
	configurator.WithChallenger(addrFor(devkeys.ChallengerRole))
}

func WithDevkeyL1Roles(t require.TestingT, dk devkeys.Keys, configurator L2Configurator, l1ChainID eth.ChainID) {
	addrFor := RoleToAddrProvider(t, dk, l1ChainID)
	configurator.WithL1ProxyAdminOwner(addrFor(devkeys.L1ProxyAdminOwnerRole))
}

func WithDevkeySuperRoles(t require.TestingT, dk devkeys.Keys, l1ID eth.ChainID, configurator SuperchainConfigurator) {
	addrFor := RoleToAddrProvider(t, dk, l1ID)
	configurator.WithGuardian(addrFor(devkeys.SuperchainConfigGuardianKey))
	configurator.WithProtocolVersionsOwner(addrFor(devkeys.SuperchainDeployerKey))
	configurator.WithProxyAdminOwner(addrFor(devkeys.L1ProxyAdminOwnerRole))
	configurator.WithChallenger(addrFor(devkeys.ChallengerRole))
}

func WithOverrideGuardianToL1PAO(t require.TestingT, dk devkeys.Keys, l1ID eth.ChainID, configurator SuperchainConfigurator) {
	addrFor := RoleToAddrProvider(t, dk, l1ID)
	configurator.WithGuardian(addrFor(devkeys.L1ProxyAdminOwnerRole))
}

func KeyToAddrProvider(t require.TestingT, dk devkeys.Keys) func(k devkeys.Key) common.Address {
	return func(k devkeys.Key) common.Address {
		addr, err := dk.Address(k)
		require.NoError(t, err, "failed to get address for key %s", k)
		return addr
	}
}

func RoleToAddrProvider(t require.TestingT, dk devkeys.Keys, chainID eth.ChainID) func(k devkeys.Role) common.Address {
	return func(role devkeys.Role) common.Address {
		k := role.Key(chainID.ToBig())
		addr, err := dk.Address(k)
		require.NoError(t, err, "failed to get address for key %s", k)
		return addr
	}
}

type intentBuilder struct {
	t                require.TestingT
	l1StartBlockHash *common.Hash
	intent           *state.Intent
}

func New() Builder {
	return &intentBuilder{
		intent: &state.Intent{
			ConfigType:      state.IntentTypeCustom,
			SuperchainRoles: new(addresses.SuperchainRoles),
		},
	}
}

func (b *intentBuilder) WithL1ContractsLocator(loc *artifacts.Locator) Builder {
	b.intent.L1ContractsLocator = loc
	return b
}

func (b *intentBuilder) WithL2ContractsLocator(loc *artifacts.Locator) Builder {
	b.intent.L2ContractsLocator = loc
	return b
}

func (b *intentBuilder) WithSuperchain() (Builder, SuperchainConfigurator) {
	return b, &superchainConfigurator{builder: b}
}

func (b *intentBuilder) WithL1(l1ChainID eth.ChainID) (Builder, L1Configurator) {
	b.intent.L1ChainID = l1ChainID.ToBig().Uint64()
	return b, &l1Configurator{builder: b}
}

func (b *intentBuilder) WithL2(l2ChainID eth.ChainID) (Builder, L2Configurator) {
	chainIntent := &state.ChainIntent{
		ID:                       common.BigToHash(l2ChainID.ToBig()),
		Eip1559DenominatorCanyon: standard.Eip1559DenominatorCanyon,
		Eip1559Denominator:       standard.Eip1559Denominator,
		Eip1559Elasticity:        standard.Eip1559Elasticity,
		DeployOverrides:          make(map[string]any),
	}
	b.intent.Chains = append(b.intent.Chains, chainIntent)
	return b, &l2Configurator{builder: b, chainIndex: len(b.intent.Chains) - 1}
}

func (b *intentBuilder) L2s() (out []L2Configurator) {
	for i := range b.intent.Chains {
		out = append(out, &l2Configurator{builder: b, chainIndex: i})
	}
	return out
}

// WithGlobalOverride sets a global override.
// This is generally discouraged, but may be needed to work around legacy configuration constraints.
func (b *intentBuilder) WithGlobalOverride(key string, value any) Builder {
	if b.intent.GlobalDeployOverrides == nil {
		b.intent.GlobalDeployOverrides = make(map[string]any)
	}
	b.intent.GlobalDeployOverrides[key] = value
	return b
}

func (b *intentBuilder) Build() (*state.Intent, error) {
	require.NoError(b.t, b.intent.Check(), "invalid intent")
	return b.intent, nil
}

type superchainConfigurator struct {
	builder *intentBuilder
}

func (c *superchainConfigurator) ID() SuperchainID {
	return "main"
}

func (c *superchainConfigurator) L1ChainID() eth.ChainID {
	return eth.ChainIDFromUInt64(c.builder.intent.L1ChainID)
}

func (c *superchainConfigurator) WithSuperchainConfigProxy(address common.Address) SuperchainConfigurator {
	c.builder.intent.SuperchainConfigProxy = &address
	return c
}

func (c *superchainConfigurator) WithProxyAdminOwner(address common.Address) SuperchainConfigurator {
	c.builder.intent.SuperchainRoles.SuperchainProxyAdminOwner = address
	return c
}

func (c *superchainConfigurator) WithGuardian(address common.Address) SuperchainConfigurator {
	c.builder.intent.SuperchainRoles.SuperchainGuardian = address
	return c
}

func (c *superchainConfigurator) WithProtocolVersionsOwner(address common.Address) SuperchainConfigurator {
	c.builder.intent.SuperchainRoles.ProtocolVersionsOwner = address
	return c
}

func (c *superchainConfigurator) WithChallenger(address common.Address) SuperchainConfigurator {
	c.builder.intent.SuperchainRoles.Challenger = address
	return c
}

type l1Configurator struct {
	builder *intentBuilder
}

func (c *l1Configurator) WithChainID(chainID eth.ChainID) L1Configurator {
	c.builder.intent.L1ChainID = chainID.ToBig().Uint64()
	return c
}

func (c *l1Configurator) initL1DevGenesisParams() {
	if c.builder.intent.L1DevGenesisParams == nil {
		c.builder.intent.L1DevGenesisParams = &state.L1DevGenesisParams{
			Prefund: make(map[common.Address]*hexutil.U256),
		}
	}
}

func (c *l1Configurator) WithTimestamp(v uint64) L1Configurator {
	c.initL1DevGenesisParams()
	c.builder.intent.L1DevGenesisParams.BlockParams.Timestamp = v
	return c
}

func (c *l1Configurator) WithGasLimit(v uint64) L1Configurator {
	c.initL1DevGenesisParams()
	c.builder.intent.L1DevGenesisParams.BlockParams.GasLimit = v
	return c
}

func (c *l1Configurator) WithExcessBlobGas(v uint64) L1Configurator {
	c.initL1DevGenesisParams()
	c.builder.intent.L1DevGenesisParams.BlockParams.ExcessBlobGas = v
	return c
}

func (c *l1Configurator) WithPragueOffset(v uint64) L1Configurator {
	c.initL1DevGenesisParams()
	c.builder.intent.L1DevGenesisParams.PragueTimeOffset = &v
	return c
}

func (c *l1Configurator) WithPrefundedAccount(addr common.Address, amount uint256.Int) L1Configurator {
	c.initL1DevGenesisParams()
	c.builder.intent.L1DevGenesisParams.Prefund[addr] = (*hexutil.U256)(&amount)
	return c
}

type l2Configurator struct {
	t          require.TestingT
	builder    *intentBuilder
	chainIndex int
}

func (c *l2Configurator) L1Config() L1Configurator {
	return &l1Configurator{builder: c.builder}
}

func (c *l2Configurator) ChainID() eth.ChainID {
	return eth.ChainIDFromBig(c.builder.intent.Chains[c.chainIndex].ID.Big())
}

func (c *l2Configurator) WithBlockTime(blockTime uint64) {
	c.builder.intent.Chains[c.chainIndex].DeployOverrides["l2BlockTime"] = blockTime
}

func (c *l2Configurator) WithL1StartBlockHash(hash common.Hash) {
	c.builder.l1StartBlockHash = &hash
}

func (c *l2Configurator) WithL1ContractsLocator(urlStr string) {
	c.builder.intent.L1ContractsLocator = artifacts.MustNewLocatorFromURL(urlStr)
}

func (c *l2Configurator) WithL2ContractsLocator(urlStr string) {
	c.builder.intent.L2ContractsLocator = artifacts.MustNewLocatorFromURL(urlStr)
}

func (c *l2Configurator) WithBaseFeeVaultRecipient(address common.Address) {
	c.builder.intent.Chains[c.chainIndex].BaseFeeVaultRecipient = address
}

func (c *l2Configurator) WithSequencerFeeVaultRecipient(address common.Address) {
	c.builder.intent.Chains[c.chainIndex].SequencerFeeVaultRecipient = address
}

func (c *l2Configurator) WithL1FeeVaultRecipient(address common.Address) {
	c.builder.intent.Chains[c.chainIndex].L1FeeVaultRecipient = address
}

func (c *l2Configurator) WithL1ProxyAdminOwner(address common.Address) {
	c.builder.intent.Chains[c.chainIndex].Roles.L1ProxyAdminOwner = address
}

func (c *l2Configurator) WithL2ProxyAdminOwner(address common.Address) {
	c.builder.intent.Chains[c.chainIndex].Roles.L2ProxyAdminOwner = address
}

func (c *l2Configurator) WithSystemConfigOwner(address common.Address) {
	c.builder.intent.Chains[c.chainIndex].Roles.SystemConfigOwner = address
}

func (c *l2Configurator) WithUnsafeBlockSigner(address common.Address) {
	c.builder.intent.Chains[c.chainIndex].Roles.UnsafeBlockSigner = address
}

func (c *l2Configurator) WithBatcher(address common.Address) {
	c.builder.intent.Chains[c.chainIndex].Roles.Batcher = address
}

func (c *l2Configurator) WithProposer(address common.Address) {
	c.builder.intent.Chains[c.chainIndex].Roles.Proposer = address
}

func (c *l2Configurator) WithChallenger(address common.Address) {
	c.builder.intent.Chains[c.chainIndex].Roles.Challenger = address
}

func (c *l2Configurator) WithEIP1559DenominatorCanyon(value uint64) {
	c.builder.intent.Chains[c.chainIndex].Eip1559DenominatorCanyon = value
}

func (c *l2Configurator) WithEIP1559Denominator(value uint64) {
	c.builder.intent.Chains[c.chainIndex].Eip1559Denominator = value
}

func (c *l2Configurator) WithEIP1559Elasticity(value uint64) {
	c.builder.intent.Chains[c.chainIndex].Eip1559Elasticity = value
}

func (c *l2Configurator) WithOperatorFeeScalar(value uint64) {
	c.builder.intent.Chains[c.chainIndex].OperatorFeeScalar = uint32(value)
}

func (c *l2Configurator) WithOperatorFeeConstant(value uint64) {
	c.builder.intent.Chains[c.chainIndex].OperatorFeeConstant = value
}

func (c *l2Configurator) WithForkAtGenesis(fork rollup.ForkName) {
	var future bool
	for _, refFork := range rollup.AllForks {
		if refFork == rollup.Bedrock {
			continue
		}

		if future {
			c.WithForkAtOffset(refFork, nil)
		} else {
			c.WithForkAtOffset(refFork, new(uint64))
		}

		if refFork == fork {
			future = true
		}
	}
}

func (c *l2Configurator) WithForkAtOffset(fork rollup.ForkName, offset *uint64) {
	require.True(c.t, rollup.IsValidFork(fork))
	key := fmt.Sprintf("l2Genesis%sTimeOffset", cases.Title(language.English).String(string(fork)))

	if offset == nil {
		delete(c.builder.intent.Chains[c.chainIndex].DeployOverrides, key)
	} else {
		// The typing is important, or op-deployer merge-JSON tricks will fail
		c.builder.intent.Chains[c.chainIndex].DeployOverrides[key] = (*hexutil.Uint64)(offset)
	}
}

func (c *l2Configurator) initL2DevGenesisParams() *state.L2DevGenesisParams {
	chainIntent := c.builder.intent.Chains[c.chainIndex]
	if chainIntent.L2DevGenesisParams == nil {
		chainIntent.L2DevGenesisParams = &state.L2DevGenesisParams{Prefund: make(map[common.Address]*hexutil.U256)}
	}
	return chainIntent.L2DevGenesisParams
}

func (c *l2Configurator) WithPrefundedAccount(addr common.Address, amount uint256.Int) L2Configurator {
	c.initL2DevGenesisParams().Prefund[addr] = (*hexutil.U256)(&amount)
	return c
}

func (c *l2Configurator) WithAdditionalDisputeGames(games []state.AdditionalDisputeGame) {
	chain := c.builder.intent.Chains[c.chainIndex]
	if chain.AdditionalDisputeGames == nil {
		chain.AdditionalDisputeGames = make([]state.AdditionalDisputeGame, 0)
	}
	chain.AdditionalDisputeGames = append(chain.AdditionalDisputeGames, games...)
}

func (c *l2Configurator) WithFinalizationPeriodSeconds(value uint64) {
	c.builder.intent.Chains[c.chainIndex].DeployOverrides["l2FinalizationPeriodSeconds"] = value
}
