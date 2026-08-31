package sysgo

import (
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/params/forks"
	"github.com/holiman/uint256"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-chain-ops/interopgen/config"
	"github.com/ethereum-optimism/optimism/op-core/devfeatures"
	opforks "github.com/ethereum-optimism/optimism/op-core/forks"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/artifacts"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/inspect"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/intentbuilder"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testreq"
)

// funderMnemonicIndex the funding account is not one of the 30 standard account, but still derived from a user-key.
const funderMnemonicIndex = 10_000

// FunderKey returns the private key of the genesis-prefunded funder account.
// Every sysgo runtime prefunds this account at genesis (see the WithPrefundedAccount
// calls in this package), so tests hand out funds from a prefunded EOA
// (dsl.NewFunderEOA) rather than a hosted faucet service. Setup transactions
// using this key must be included before NewFunderEOA snapshots its nonce; the
// FunderEOA must own the key exclusively thereafter.
func FunderKey(keys devkeys.Keys) (*ecdsa.PrivateKey, error) {
	return keys.Secret(devkeys.UserKey(funderMnemonicIndex))
}

const devFeatureBitmapKey = "devFeatureBitmap"
const DevstackL1ForkEnvVar = "DEVSTACK_L1_FORK"

// proxyImplementationSlot is the EIP-1967 proxy implementation storage slot used
// by every L2 predeploy proxy (`bytes32(uint256(keccak256("eip1967.proxy.implementation")) - 1)`).
// Mirrors Constants.PROXY_IMPLEMENTATION_ADDRESS in packages/contracts-bedrock.
var proxyImplementationSlot = common.HexToHash("0x360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc")

type DeployerOption func(p devtest.T, keys devkeys.Keys, builder intentbuilder.Builder)

func WithForkAtL1Genesis(fork forks.Fork) DeployerOption {
	return func(_ devtest.T, _ devkeys.Keys, builder intentbuilder.Builder) {
		builder.L1().WithL1ForkAtGenesis(fork)
	}
}

func WithForkAtL1Offset(fork forks.Fork, offset uint64) DeployerOption {
	return func(_ devtest.T, _ devkeys.Keys, builder intentbuilder.Builder) {
		builder.L1().WithL1ForkAtOffset(fork, &offset)
	}
}

func defaultL1BlobSchedule() *params.BlobScheduleConfig {
	return &params.BlobScheduleConfig{
		Cancun: params.DefaultCancunBlobConfig,
		Osaka:  params.DefaultOsakaBlobConfig,
		Prague: params.DefaultPragueBlobConfig,
		BPO1:   params.DefaultBPO1BlobConfig,
		BPO2:   params.DefaultBPO2BlobConfig,
		BPO3:   params.DefaultBPO3BlobConfig,
		BPO4:   params.DefaultBPO4BlobConfig,
		// Upstream defaults are not available yet, so keep the latest parameters.
		BPO5:      params.DefaultBPO4BlobConfig,
		Amsterdam: params.DefaultBPO4BlobConfig,
	}
}

func WithDefaultBPOBlobSchedule(_ devtest.T, _ devkeys.Keys, builder intentbuilder.Builder) {
	builder.L1().WithL1BlobSchedule(defaultL1BlobSchedule())
}

// parseL1Fork accepts both Ethereum upgrade names and their geth execution-layer names.
// BPO selectors are intentionally excluded while the bundled op-geth rejects newPayloadV4 for them.
func parseL1Fork(value string) (forks.Fork, error) {
	fork, ok := map[string]forks.Fork{
		"pectra":      forks.Prague,
		"prague":      forks.Prague,
		"fusaka":      forks.Osaka,
		"osaka":       forks.Osaka,
		"glamsterdam": forks.Amsterdam,
		"amsterdam":   forks.Amsterdam,
	}[value]
	if ok {
		return fork, nil
	}
	return 0, fmt.Errorf("unsupported L1 fork %q", value)
}

func WithKarstAtOffset(offset *uint64) DeployerOption {
	return func(p devtest.T, _ devkeys.Keys, builder intentbuilder.Builder) {
		for _, l2Cfg := range builder.L2s() {
			l2Cfg.WithForkAtOffset(opforks.Karst, offset)
		}
	}
}

func WithKarstAtGenesis(p devtest.T, _ devkeys.Keys, builder intentbuilder.Builder) {
	for _, l2Cfg := range builder.L2s() {
		l2Cfg.WithForkAtGenesis(opforks.Karst)
	}
}

// WithKeepKarstUpgradeGas opts the L2 chains out of reverting the Karst activation block's
// one-time upgrade gas, so post-activation blocks keep the inflated gas limit.
func WithKeepKarstUpgradeGas(p devtest.T, _ devkeys.Keys, builder intentbuilder.Builder) {
	for _, l2Cfg := range builder.L2s() {
		l2Cfg.WithKeepKarstUpgradeGas()
	}
}

func WithJovianAtGenesis(p devtest.T, _ devkeys.Keys, builder intentbuilder.Builder) {
	for _, l2Cfg := range builder.L2s() {
		l2Cfg.WithForkAtGenesis(opforks.Jovian)
	}
}

func WithL2GasLimit(gasLimit uint64) DeployerOption {
	return func(_ devtest.T, _ devkeys.Keys, builder intentbuilder.Builder) {
		for _, l2Cfg := range builder.L2s() {
			l2Cfg.WithGasLimit(gasLimit)
		}
	}
}

func WithEcotoneAtGenesis(p devtest.T, _ devkeys.Keys, builder intentbuilder.Builder) {
	for _, l2Cfg := range builder.L2s() {
		l2Cfg.WithForkAtGenesis(opforks.Ecotone)
	}
}

type DeployerPipelineOption func(wb *worldBuilder, intent *state.Intent, cfg *deployer.ApplyPipelineOpts)

func WithDeployerCacheDir(dirPath string) DeployerPipelineOption {
	return func(_ *worldBuilder, _ *state.Intent, cfg *deployer.ApplyPipelineOpts) {
		cfg.CacheDir = dirPath
	}
}

// WithDAFootprintGasScalar sets the DA footprint gas scalar with which the networks identified by
// l2ChainIDs will be launched. If there are no l2ChainIDs provided, all L2 networks are set with scalar.
func WithDAFootprintGasScalar(scalar uint16, l2ChainIDs ...eth.ChainID) DeployerOption {
	return func(p devtest.T, _ devkeys.Keys, builder intentbuilder.Builder) {
		for _, l2 := range builder.L2s() {
			if len(l2ChainIDs) == 0 || slices.Contains(l2ChainIDs, l2.ChainID()) {
				l2.WithDAFootprintGasScalar(scalar)
			}
		}
	}
}

type L2Deployment struct {
	systemConfigProxyAddr          common.Address
	disputeGameFactoryProxy        common.Address
	l1StandardBridgeProxy          common.Address
	proxyAdmin                     common.Address
	permissionlessDelayedWETHProxy common.Address
}

var _ stack.L2Deployment = &L2Deployment{}

func (d *L2Deployment) SystemConfigProxyAddr() common.Address {
	return d.systemConfigProxyAddr
}

func (d *L2Deployment) DisputeGameFactoryProxyAddr() common.Address {
	return d.disputeGameFactoryProxy
}

func (d *L2Deployment) L1StandardBridgeProxyAddr() common.Address {
	return d.l1StandardBridgeProxy
}

func (d *L2Deployment) ProxyAdminAddr() common.Address {
	return d.proxyAdmin
}

func (d *L2Deployment) PermissionlessDelayedWETHProxyAddr() common.Address {
	return d.permissionlessDelayedWETHProxy
}

type worldBuilder struct {
	p devtest.CommonT

	logger  log.Logger
	require *testreq.Assertions
	keys    devkeys.Keys

	// options
	deployerPipelineOptions []DeployerPipelineOption

	// preForkPredeployAllocs, when non-nil, is overlaid onto every L2 chain's
	// genesis predeploy accounts before the genesis and rollup config are built.
	preForkPredeployAllocs types.GenesisAlloc

	builder intentbuilder.Builder

	// deployerAddr and deployerCacheDir are the two pipeline inputs a later, partial re-run of one
	// stage needs and cannot recover from the state: the address scripts are pranked as, and where
	// the contract artifacts were resolved to. A private interop pair re-runs the L2-genesis stage
	// to produce its second half (see private_interop_genesis.go).
	deployerAddr     common.Address
	deployerCacheDir string

	output          *state.State
	outL1Genesis    *core.Genesis
	l2Chains        []eth.ChainID
	outL2Genesis    map[eth.ChainID]*core.Genesis
	outL2RollupCfg  map[eth.ChainID]*rollup.Config
	outL2Deployment map[eth.ChainID]*L2Deployment

	outFullCfgSet config.FullConfigSetMerged

	outSuperchainDeployment *SuperchainDeployment
}

var (
	oneEth     = uint256.NewInt(1e18)
	millionEth = new(uint256.Int).Mul(uint256.NewInt(1e6), oneEth)
)

func WithEmbeddedContractSources() DeployerOption {
	return func(_ devtest.T, _ devkeys.Keys, builder intentbuilder.Builder) {
		setContractLocators(builder, artifacts.EmbeddedLocator)
	}
}

func localContractSourcesLocator(artifactsPath string) (*artifacts.Locator, error) {
	if artifactsPath == "" {
		paths, err := contractPaths()
		if err != nil {
			return nil, err
		}
		artifactsPath = paths.FoundryArtifacts
	}
	absPath, err := filepath.Abs(artifactsPath)
	if err != nil {
		return nil, err
	}
	if err := ensureDir(absPath); err != nil {
		return nil, err
	}
	return artifacts.NewFileLocator(absPath)
}

func setContractLocators(builder intentbuilder.Builder, contractArtifacts *artifacts.Locator) {
	builder.WithL1ContractsLocator(contractArtifacts)
	builder.WithL2ContractsLocator(contractArtifacts)
}

// WithLocalContractSourcesAt configures the deployer to load both L1 and L2
// contract artifacts from the given local contracts-bedrock checkout or
// forge-artifacts directory.
func WithLocalContractSourcesAt(artifactsPath string) DeployerOption {
	return func(p devtest.T, _ devkeys.Keys, builder intentbuilder.Builder) {
		contractArtifacts, err := localContractSourcesLocator(artifactsPath)
		p.Require().NoError(err)
		setContractLocators(builder, contractArtifacts)
	}
}

func WithLocalContractSources() DeployerOption {
	return func(p devtest.T, _ devkeys.Keys, builder intentbuilder.Builder) {
		contractArtifacts, err := localContractSourcesLocator("")
		p.Require().NoError(err)
		setContractLocators(builder, contractArtifacts)
	}
}

func WithCommons(l1ChainID eth.ChainID) DeployerOption {
	return func(p devtest.T, keys devkeys.Keys, builder intentbuilder.Builder) {
		_, l1Config := builder.WithL1(l1ChainID)

		l1StartTimestamp := uint64(time.Now().Unix()) + 1
		l1Config.WithTimestamp(l1StartTimestamp)

		l1Fork := forks.Prague // activate Pectra on L1 by default
		if value, ok := os.LookupEnv(DevstackL1ForkEnvVar); ok {
			var err error
			l1Fork, err = parseL1Fork(value)
			p.Require().NoError(err, "invalid %s", DevstackL1ForkEnvVar)
		}
		l1Config.WithL1BlobSchedule(defaultL1BlobSchedule())
		l1Config.WithL1ForkAtGenesis(l1Fork)

		funderAddr, err := keys.Address(devkeys.UserKey(funderMnemonicIndex))
		p.Require().NoError(err, "need funder addr")
		l1Config.WithPrefundedAccount(funderAddr, *eth.BillionEther.ToU256())

		// We use the L1 chain ID to identify the superchain-wide roles.
		addrFor := intentbuilder.RoleToAddrProvider(p, keys, l1ChainID)
		_, superCfg := builder.WithSuperchain()
		intentbuilder.WithDevkeySuperRoles(p, keys, l1ChainID, superCfg)
		l1Config.WithPrefundedAccount(addrFor(devkeys.SuperchainProxyAdminOwner), *millionEth)
		l1Config.WithPrefundedAccount(addrFor(devkeys.SuperchainConfigGuardianKey), *millionEth)
		l1Config.WithPrefundedAccount(addrFor(devkeys.L1ProxyAdminOwnerRole), *millionEth)
	}
}

func WithGuardianMatchL1PAO() DeployerOption {
	return func(p devtest.T, keys devkeys.Keys, builder intentbuilder.Builder) {
		_, superCfg := builder.WithSuperchain()
		intentbuilder.WithOverrideGuardianToL1PAO(p, keys, superCfg.L1ChainID(), superCfg)
	}
}

func WithPrefundedL2(l1ChainID, l2ChainID eth.ChainID) DeployerOption {
	return func(p devtest.T, keys devkeys.Keys, builder intentbuilder.Builder) {
		_, l2Config := builder.WithL2(l2ChainID)
		intentbuilder.WithDevkeyVaults(p, keys, l2Config)
		intentbuilder.WithDevkeyL2Roles(p, keys, l2Config)
		// l2configurator L1ProxyAdminOwner must be also populated
		intentbuilder.WithDevkeyL1Roles(p, keys, l2Config, l1ChainID)
		{
			funderAddr, err := keys.Address(devkeys.UserKey(funderMnemonicIndex))
			p.Require().NoError(err, "need funder addr")
			l2Config.WithPrefundedAccount(funderAddr, *eth.BillionEther.ToU256())
		}
		{
			addrFor := intentbuilder.RoleToAddrProvider(p, keys, l2ChainID)
			l1Config := l2Config.L1Config()
			l1Config.WithPrefundedAccount(addrFor(devkeys.BatcherRole), *millionEth)
			l1Config.WithPrefundedAccount(addrFor(devkeys.ProposerRole), *millionEth)
			l1Config.WithPrefundedAccount(addrFor(devkeys.ChallengerRole), *millionEth)
			l1Config.WithPrefundedAccount(addrFor(devkeys.SystemConfigOwner), *millionEth)
		}
	}
}

// WithDevFeatureEnabled adds a feature as enabled in the dev feature bitmap
func WithDevFeatureEnabled(flag common.Hash) DeployerOption {
	return func(p devtest.T, keys devkeys.Keys, builder intentbuilder.Builder) {
		currentValue := builder.GlobalOverride(devFeatureBitmapKey)
		var bitmap common.Hash
		if currentValue != nil {
			bitmap = currentValue.(common.Hash)
		}
		builder.WithGlobalOverride(devFeatureBitmapKey, devfeatures.EnableDevFeature(bitmap, flag))
		if flag == devfeatures.OptimismPortalInteropFlag {
			builder.WithUseInterop(true)
		}
	}
}

// WithInteropAtGenesis activates interop at genesis for all known L2s
func WithInteropAtGenesis() DeployerOption {
	return func(p devtest.T, keys devkeys.Keys, builder intentbuilder.Builder) {
		for _, l2Cfg := range builder.L2s() {
			l2Cfg.WithForkAtGenesis(opforks.Lagoon)
		}
	}
}

// WithHardforkSequentialActivation configures a deployment such that L2 chains
// activate hardforks sequentially, starting from startFork and continuing
// until (including) endFork. Each successive fork is scheduled at
// an increasing offset.
func WithHardforkSequentialActivation(startFork, endFork opforks.Name, delta *uint64) DeployerOption {
	return func(p devtest.T, keys devkeys.Keys, builder intentbuilder.Builder) {
		for _, l2Cfg := range builder.L2s() {
			l2Cfg.WithForkAtGenesis(startFork)
			activateWithOffset := false
			deactivate := false
			for idx, refFork := range opforks.All {
				if deactivate {
					l2Cfg.WithForkAtOffset(refFork, nil)
					deactivate = true
					continue
				}
				if activateWithOffset {
					offset := *delta * uint64(idx)
					l2Cfg.WithForkAtOffset(refFork, &offset)
				}
				if startFork == refFork {
					activateWithOffset = true
				}
				if endFork == refFork {
					deactivate = true
				}
			}
		}
	}
}

// WithSequencingWindow overrides the number of L1 blocks in a sequencing window, applied to all L2s.
func WithSequencingWindow(n uint64) DeployerOption {
	return func(p devtest.T, keys devkeys.Keys, builder intentbuilder.Builder) {
		builder.WithGlobalOverride("sequencerWindowSize", uint64(n))
	}
}

func WithDeployerMatchL1PAO() DeployerPipelineOption {
	return func(wb *worldBuilder, intent *state.Intent, cfg *deployer.ApplyPipelineOpts) {
		l1ChainID := new(big.Int).SetUint64(intent.L1ChainID)
		deployerKey, err := wb.keys.Secret(devkeys.L1ProxyAdminOwnerRole.Key(l1ChainID))
		wb.require.NoError(err)
		cfg.DeployerPrivateKey = deployerKey
	}
}

// WithL2BlockTimes sets per-chain L2 block times. The map keys are L2 chain
// IDs and values are the desired block time in seconds for that chain.
func WithL2BlockTimes(blockTimes map[eth.ChainID]uint64) DeployerOption {
	return func(_ devtest.T, _ devkeys.Keys, builder intentbuilder.Builder) {
		for _, l2Cfg := range builder.L2s() {
			if bt, ok := blockTimes[l2Cfg.ChainID()]; ok {
				l2Cfg.WithBlockTime(bt)
			}
		}
	}
}

// WithUniformL2BlockTimes sets the same L2 block time (in seconds) on every
// configured L2 chain.
func WithUniformL2BlockTimes(seconds uint64) DeployerOption {
	return func(_ devtest.T, _ devkeys.Keys, builder intentbuilder.Builder) {
		for _, l2Cfg := range builder.L2s() {
			l2Cfg.WithBlockTime(seconds)
		}
	}
}

// WithFinalizationPeriodSeconds overrides the number of L1 blocks in a sequencing window, applied to all L2s.
func WithFinalizationPeriodSeconds(n uint64) DeployerOption {
	return func(p devtest.T, keys devkeys.Keys, builder intentbuilder.Builder) {
		for _, l2Cfg := range builder.L2s() {
			l2Cfg.WithFinalizationPeriodSeconds(n)
		}
	}
}

func WithProofMaturityDelaySeconds(n uint64) DeployerOption {
	return func(p devtest.T, keys devkeys.Keys, builder intentbuilder.Builder) {
		builder.WithGlobalOverride("proofMaturityDelaySeconds", uint64(n))
	}
}

func WithDisputeGameFinalityDelaySeconds(seconds uint64) DeployerOption {
	return func(p devtest.T, keys devkeys.Keys, builder intentbuilder.Builder) {
		builder.WithGlobalOverride("disputeGameFinalityDelaySeconds", seconds)
	}
}

func WithCustomGasToken(name, symbol string, initialLiquidity *big.Int, liquidityControllerOwner common.Address) DeployerOption {
	return func(p devtest.T, keys devkeys.Keys, builder intentbuilder.Builder) {
		for _, l2Cfg := range builder.L2s() {
			l2Cfg.WithCustomGasToken(name, symbol, initialLiquidity, liquidityControllerOwner)
		}
	}
}

// WithCustomGasTokenOn enables the custom gas token on ONE chain and leaves every other L2 in the
// intent paying in ETH.
//
// The op-deployer intent has always been per-chain -- CustomGasToken is a field on ChainIntent --
// but the only devstack door to it fans out over builder.L2s(), so a preset could ask for "custom
// gas token" or "no custom gas token" and nothing in between. A private interop pair needs exactly
// the in-between: the private chain IS a custom gas token chain and its interop counterparty is
// not, and neither is the private chain's own public rendering, whose replay transactions pay gas
// in the rendering's own ETH.
func WithCustomGasTokenOn(
	chainID eth.ChainID,
	name, symbol string,
	initialLiquidity *big.Int,
	liquidityControllerOwner common.Address,
) DeployerOption {
	return func(p devtest.T, keys devkeys.Keys, builder intentbuilder.Builder) {
		l2Cfg := findL2(p, builder, chainID)
		l2Cfg.WithCustomGasToken(name, symbol, initialLiquidity, liquidityControllerOwner)
	}
}

// WithPrivateInterop marks ONE chain in the intent as a half of a private interop pair, so its
// genesis renders that half. Ordinary chains in the same intent are untouched.
func WithPrivateInterop(chainID eth.ChainID, cfg *state.PrivateInterop) DeployerOption {
	return func(p devtest.T, keys devkeys.Keys, builder intentbuilder.Builder) {
		findL2(p, builder, chainID).WithPrivateInterop(cfg)
	}
}

// findL2 returns the configurator for one chain, failing loudly rather than silently doing nothing
// when the intent has no such chain -- a per-chain option that matches nothing is the failure mode
// worth catching.
func findL2(p devtest.T, builder intentbuilder.Builder, chainID eth.ChainID) intentbuilder.L2Configurator {
	for _, l2Cfg := range builder.L2s() {
		if l2Cfg.ChainID() == chainID {
			return l2Cfg
		}
	}
	p.Require().FailNow("no L2 with chain ID " + chainID.String() + " in the intent")
	return nil
}

func (wb *worldBuilder) buildL1Genesis() {
	wb.require.NotNil(wb.output.L1DevGenesis, "must have L1 genesis outer config")
	wb.require.NotNil(wb.output.L1StateDump, "must have L1 genesis alloc")

	genesisOuter := wb.output.L1DevGenesis
	genesisAlloc := wb.output.L1StateDump.Data.Accounts
	genesisCfg := *genesisOuter
	genesisCfg.StateHash = nil
	genesisCfg.Alloc = genesisAlloc

	wb.outL1Genesis = &genesisCfg
}

func (wb *worldBuilder) buildL2Genesis() {
	wb.outL2Genesis = make(map[eth.ChainID]*core.Genesis)
	wb.outL2RollupCfg = make(map[eth.ChainID]*rollup.Config)
	for _, ch := range wb.output.Chains {
		if wb.preForkPredeployAllocs != nil {
			wb.require.NotNil(ch.Allocs, "chain must have allocs to overlay pre-fork state onto")
			for addr, acct := range wb.preForkPredeployAllocs {
				// Permit2's bytecode contains chain-id-derived immutables (cached
				// domain separator), so the frozen snapshot's Permit2 is only
				// valid for its source chain. Keep each chain's own Permit2.
				// Other chain-agnostic preinstalls are safe to overlay wholesale.
				if addr == predeploys.Permit2Addr {
					continue
				}
				// Proxied predeploys have chain-specific storage (owners,
				// balances, mappings, etc.) that the frozen snapshot would
				// clobber. For these, only overlay the EIP-1967 implementation
				// slot so the proxy delegates to the frozen pre-fork
				// implementation while keeping each chain's own state.
				// Non-proxied predeploys (WETH, GovernanceToken, etc.) and
				// non-predeploy entries (implementation contracts at 0xc0d3…,
				// preinstalls, deployer EOA, …) are overlaid wholesale.
				if p, ok := predeploys.PredeploysByAddress[addr]; ok && !p.ProxyDisabled {
					existing, ok := ch.Allocs.Data.Accounts[addr]
					wb.require.Truef(ok, "predeploy %s missing from chain genesis allocs", addr)
					implSlot, ok := acct.Storage[proxyImplementationSlot]
					if !ok {
						// No pre-fork implementation to pin: this proxy had no
						// implementation in the frozen state. Leave
						// the chain's own proxy state; the fork's NUT bundle
						// installs the implementation at activation.
						continue
					}
					if existing.Storage == nil {
						existing.Storage = make(map[common.Hash]common.Hash, 1)
					}
					existing.Storage[proxyImplementationSlot] = implSlot
					ch.Allocs.Data.Accounts[addr] = existing
					continue
				}
				ch.Allocs.Data.Accounts[addr] = acct
			}
		}
		l2Genesis, l2RollupCfg, err := inspect.GenesisAndRollup(wb.output, ch.ID)
		wb.require.NoError(err, "need L2 genesis and rollup")
		id := eth.ChainIDFromBytes32(ch.ID)
		wb.outL2Genesis[id] = l2Genesis
		wb.outL2RollupCfg[id] = l2RollupCfg
		// op-geth is deprecated as of the Karst fork and refuses to build, seal,
		// or import Karst blocks. The op-geth EL lane therefore cannot run any
		// chain that ever activates Karst, so skip such tests here — the single
		// point every runtime builds genesis through — letting Karst+ coverage
		// run on op-reth (the official Karst EL client). op-geth still covers
		// chains up to Jovian.
		if l2Genesis.Config.KarstTime != nil && devstackL2ELKind() == MixedL2ELOpGeth {
			wb.p.Logf("op-geth is deprecated as of Karst; skipping test: chain %s activates Karst and DEVSTACK_L2EL_KIND=op-geth", id)
			wb.p.SkipNow()
		}
	}
}

func (wb *worldBuilder) buildL2DeploymentOutputs() {
	wb.outL2Deployment = make(map[eth.ChainID]*L2Deployment)
	for _, ch := range wb.output.Chains {
		chainID := eth.ChainIDFromBytes32(ch.ID)
		wb.outL2Deployment[chainID] = &L2Deployment{
			systemConfigProxyAddr:          ch.SystemConfigProxy,
			disputeGameFactoryProxy:        ch.DisputeGameFactoryProxy,
			l1StandardBridgeProxy:          ch.L1StandardBridgeProxy,
			proxyAdmin:                     ch.OpChainProxyAdminImpl,
			permissionlessDelayedWETHProxy: ch.DelayedWethPermissionlessGameProxy,
		}
	}
	wb.outSuperchainDeployment = &SuperchainDeployment{
		superchainConfigAddr: wb.output.SuperchainDeployment.SuperchainConfigProxy,
	}
}

func (wb *worldBuilder) buildFullConfigSet() {
	// If no chain has interop active, the dep set will be nil here,
	// so we should skip building the full config set.
	if wb.output.InteropDepSet == nil {
		return
	}

	rollupConfigSet := config.StaticRollupConfigSetFromRollupConfigMap(wb.outL2RollupCfg,
		config.StaticTimestamp(wb.outL1Genesis.Timestamp))
	fullCfgSet, err := config.NewFullConfigSetMerged(rollupConfigSet, wb.output.InteropDepSet)
	wb.require.NoError(err)
	wb.outFullCfgSet = fullCfgSet
}

func (wb *worldBuilder) Build() {
	st := &state.State{
		Version: 1,
	}

	// Work-around of op-deployer design issue.
	// We use the same deployer key for all L1 and L2 chains we deploy here.
	deployerKey, err := wb.keys.Secret(devkeys.DeployerRole.Key(big.NewInt(0)))
	wb.require.NoError(err, "need deployer key")

	intent, err := wb.builder.Build()
	wb.require.NoError(err)

	pipelineOpts := deployer.ApplyPipelineOpts{
		DeploymentTarget:   deployer.DeploymentTargetGenesis,
		L1RPCUrl:           "",
		DeployerPrivateKey: deployerKey,
		Intent:             intent,
		State:              st,
		Logger:             wb.logger,
		StateWriter:        wb, // direct output back here
		// Devstack deliberately uses an accept-all raw verifier when ZK dispute games are enabled.
		DeployMockSP1Verifier: true,
	}
	for _, opt := range wb.deployerPipelineOptions {
		opt(wb, intent, &pipelineOpts)
	}
	wb.deployerAddr = crypto.PubkeyToAddress(deployerKey.PublicKey)
	wb.deployerCacheDir = pipelineOpts.CacheDir

	err = deployer.ApplyPipeline(wb.p.Ctx(), pipelineOpts)
	wb.require.NoError(err)

	wb.require.NotNil(wb.output, "expected state-write to output")

	for _, id := range wb.output.Chains {
		chainID := eth.ChainIDFromBytes32(id.ID)
		wb.l2Chains = append(wb.l2Chains, chainID)
	}

	wb.buildL1Genesis()
	wb.buildL2Genesis()
	wb.buildL2DeploymentOutputs()
	wb.buildFullConfigSet()
}

// WriteState is a callback used by deployer.ApplyPipeline to write the output
func (wb *worldBuilder) WriteState(st *state.State) error {
	wb.output = st
	return nil
}
