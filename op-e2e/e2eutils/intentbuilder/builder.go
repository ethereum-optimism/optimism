package intentbuilder

import (
	"fmt"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/artifacts"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/standard"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type SuperchainID string

type L1Configurator interface {
	WithChainID(chainID eth.ChainID) L1Configurator
	WithStartTimestamp(timestamp uint64) L1Configurator
}

type SuperchainConfigurator interface {
	ID() SuperchainID
	WithSuperchainConfigProxy(address common.Address) SuperchainConfigurator
	WithProxyAdminOwner(address common.Address) SuperchainConfigurator
	WithGuardian(address common.Address) SuperchainConfigurator
	WithProtocolVersionsOwner(address common.Address) SuperchainConfigurator
}

type L2Configurator interface {
	ChainID() eth.ChainID
	WithBlockTime(uint64)
	WithL1StartBlockHash(hash common.Hash)
	ContractsConfigurator
	L2VaultsConfigurator
	L2RolesConfigurator
	L2FeesConfigurator
	L2HardforkConfigurator
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
	WithSuperchain() (Builder, SuperchainConfigurator)
	WithL1(l1ChainID eth.ChainID) (Builder, L1Configurator)
	WithL2(l2ChainID eth.ChainID) (Builder, L2Configurator)
	Build() (*state.Intent, error)
}

func WithDevkeyVaults(t require.TestingT, dk devkeys.Keys, configurator L2Configurator) {
	addrFor := addrProvider(t, dk, configurator.ChainID())
	configurator.WithBaseFeeVaultRecipient(addrFor(devkeys.BaseFeeVaultRecipientRole))
	configurator.WithSequencerFeeVaultRecipient(addrFor(devkeys.SequencerFeeVaultRecipientRole))
	configurator.WithL1FeeVaultRecipient(addrFor(devkeys.L1FeeVaultRecipientRole))
}

func WithDevkeyRoles(t require.TestingT, dk devkeys.Keys, configurator L2Configurator) {
	addrFor := addrProvider(t, dk, configurator.ChainID())
	configurator.WithL1ProxyAdminOwner(addrFor(devkeys.L1ProxyAdminOwnerRole))
	configurator.WithL2ProxyAdminOwner(addrFor(devkeys.L2ProxyAdminOwnerRole))
	configurator.WithSystemConfigOwner(addrFor(devkeys.SystemConfigOwner))
	configurator.WithUnsafeBlockSigner(addrFor(devkeys.SequencerP2PRole))
	configurator.WithBatcher(addrFor(devkeys.BatcherRole))
	configurator.WithProposer(addrFor(devkeys.ProposerRole))
	configurator.WithChallenger(addrFor(devkeys.ChallengerRole))
}

// addrProvider returns a function that generates addresses for a specific devkeys.Keys and chainID
func addrProvider(t require.TestingT, dk devkeys.Keys, chainID eth.ChainID) func(role devkeys.Role) common.Address {
	return func(role devkeys.Role) common.Address {
		key := role.Key(chainID.ToBig())
		addr, err := dk.Address(key)
		require.NoError(t, err, "failed to get address for role %s", role)
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
			SuperchainRoles: new(state.SuperchainRoles),
		},
	}
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

func (c *superchainConfigurator) WithSuperchainConfigProxy(address common.Address) SuperchainConfigurator {
	c.builder.intent.SuperchainConfigProxy = &address
	return c
}

func (c *superchainConfigurator) WithProxyAdminOwner(address common.Address) SuperchainConfigurator {
	c.builder.intent.SuperchainRoles.ProxyAdminOwner = address
	return c
}

func (c *superchainConfigurator) WithGuardian(address common.Address) SuperchainConfigurator {
	c.builder.intent.SuperchainRoles.Guardian = address
	return c
}

func (c *superchainConfigurator) WithProtocolVersionsOwner(address common.Address) SuperchainConfigurator {
	c.builder.intent.SuperchainRoles.ProtocolVersionsOwner = address
	return c
}

type l1Configurator struct {
	builder *intentBuilder
}

func (c *l1Configurator) WithChainID(chainID eth.ChainID) L1Configurator {
	c.builder.intent.L1ChainID = chainID.ToBig().Uint64()
	return c
}

func (c *l1Configurator) WithStartTimestamp(timestamp uint64) L1Configurator {
	c.builder.intent.L1StartTimestamp = &timestamp
	return c
}

type l2Configurator struct {
	t          require.TestingT
	builder    *intentBuilder
	chainIndex int
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
		c.builder.intent.Chains[c.chainIndex].DeployOverrides[key] = offset
	}
}
