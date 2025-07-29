package sysgo

import (
	"math/big"
	"os"
	"path/filepath"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/log"
	"github.com/holiman/uint256"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
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
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/depset"
)

// funderMnemonicIndex the funding account is not one of the 30 standard account, but still derived from a user-key.
const funderMnemonicIndex = 10_000

type DeployerOption func(p devtest.P, keys devkeys.Keys, builder intentbuilder.Builder)

func WithDeployerOptions(opts ...DeployerOption) stack.Option[*Orchestrator] {
	return stack.BeforeDeploy(func(o *Orchestrator) {
		o.P().Require().NotNil(o.wb, "must have a world builder")
		for _, opt := range opts {
			opt(o.P(), o.keys, o.wb.builder)
		}
	})
}

type DeployerPipelineOption func(wb *worldBuilder, intent *state.Intent, cfg *deployer.ApplyPipelineOpts)

func WithDeployerPipelineOption(opt DeployerPipelineOption) stack.Option[*Orchestrator] {
	return stack.BeforeDeploy(func(o *Orchestrator) {
		o.deployerPipelineOptions = append(o.deployerPipelineOptions, opt)
	})
}

func WithDeployer() stack.Option[*Orchestrator] {
	return stack.FnOption[*Orchestrator]{
		BeforeDeployFn: func(o *Orchestrator) {
			o.P().Require().Nil(o.wb, "must not already have a world builder")
			o.wb = &worldBuilder{
				p:       o.P(),
				logger:  o.P().Logger(),
				require: o.P().Require(),
				keys:    o.keys,
				builder: intentbuilder.New(),
			}
		},
		DeployFn: func(o *Orchestrator) {
			o.P().Require().NotNil(o.wb, "must have a world builder")
			o.wb.deployerPipelineOptions = o.deployerPipelineOptions
			o.wb.Build()
		},
		AfterDeployFn: func(o *Orchestrator) {
			wb := o.wb
			require := o.P().Require()
			require.NotNil(o.wb, "must have a world builder")

			l1ID := stack.L1NetworkID(eth.ChainIDFromUInt64(wb.output.AppliedIntent.L1ChainID))
			superchainID := stack.SuperchainID("main")
			clusterID := stack.ClusterID("main")

			l1Net := &L1Network{
				id:        l1ID,
				genesis:   wb.outL1Genesis,
				blockTime: 6,
			}
			o.l1Nets.Set(l1ID.ChainID(), l1Net)

			o.superchains.Set(superchainID, &Superchain{
				id:         superchainID,
				deployment: wb.outSuperchainDeployment,
			})
			o.clusters.Set(clusterID, &Cluster{
				id:     clusterID,
				cfgset: wb.outFullCfgSet,
			})

			for _, chainID := range wb.l2Chains {
				l2Genesis, ok := wb.outL2Genesis[chainID]
				require.True(ok, "L2 genesis must exist")
				l2RollupCfg, ok := wb.outL2RollupCfg[chainID]
				require.True(ok, "L2 rollup config must exist")
				l2Dep, ok := wb.outL2Deployment[chainID]
				require.True(ok, "L2 deployment must exist")

				l2ID := stack.L2NetworkID(chainID)
				l2Net := &L2Network{
					id:         l2ID,
					l1ChainID:  l1ID.ChainID(),
					genesis:    l2Genesis,
					rollupCfg:  l2RollupCfg,
					deployment: l2Dep,
					keys:       o.keys,
				}
				o.l2Nets.Set(l2ID.ChainID(), l2Net)
			}
		},
	}
}

type L2Deployment struct {
	systemConfigProxyAddr   common.Address
	disputeGameFactoryProxy common.Address
	l1StandardBridgeProxy   common.Address
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

type InteropMigration struct {
	DisputeGameFactory common.Address
}

type worldBuilder struct {
	p devtest.P

	logger  log.Logger
	require *testreq.Assertions
	keys    devkeys.Keys

	// options
	deployerPipelineOptions []DeployerPipelineOption

	builder intentbuilder.Builder

	output          *state.State
	outL1Genesis    *core.Genesis
	l2Chains        []eth.ChainID
	outL2Genesis    map[eth.ChainID]*core.Genesis
	outL2RollupCfg  map[eth.ChainID]*rollup.Config
	outL2Deployment map[eth.ChainID]*L2Deployment

	outFullCfgSet depset.FullConfigSetMerged

	outSuperchainDeployment *SuperchainDeployment

	outInteropMigration *InteropMigration
}

var (
	oneEth     = uint256.NewInt(1e18)
	millionEth = new(uint256.Int).Mul(uint256.NewInt(1e6), oneEth)
)

func WithLocalContractSources() DeployerOption {
	return func(p devtest.P, keys devkeys.Keys, builder intentbuilder.Builder) {
		paths, err := contractPaths()
		p.Require().NoError(err)
		wd, err := os.Getwd()
		p.Require().NoError(err)
		artifactsPath := filepath.Join(wd, paths.FoundryArtifacts)
		p.Require().NoError(ensureDir(artifactsPath))
		contractArtifacts, err := artifacts.NewFileLocator(artifactsPath)
		p.Require().NoError(err)
		builder.WithL1ContractsLocator(contractArtifacts)
		builder.WithL2ContractsLocator(contractArtifacts)
	}
}

func WithCommons(l1ChainID eth.ChainID) DeployerOption {
	return func(p devtest.P, keys devkeys.Keys, builder intentbuilder.Builder) {
		_, l1Config := builder.WithL1(l1ChainID)

		l1StartTimestamp := uint64(time.Now().Unix()) + 1
		l1Config.WithTimestamp(l1StartTimestamp)

		l1Config.WithPragueOffset(0) // activate pectra on L1

		faucetFunderAddr, err := keys.Address(devkeys.UserKey(funderMnemonicIndex))
		p.Require().NoError(err, "need funder addr")
		l1Config.WithPrefundedAccount(faucetFunderAddr, *eth.BillionEther.ToU256())

		// We use the L1 chain ID to identify the superchain-wide roles.
		addrFor := intentbuilder.RoleToAddrProvider(p, keys, l1ChainID)
		_, superCfg := builder.WithSuperchain()
		intentbuilder.WithDevkeySuperRoles(p, keys, l1ChainID, superCfg)
		l1Config.WithPrefundedAccount(addrFor(devkeys.SuperchainProxyAdminOwner), *millionEth)
		l1Config.WithPrefundedAccount(addrFor(devkeys.SuperchainProtocolVersionsOwner), *millionEth)
		l1Config.WithPrefundedAccount(addrFor(devkeys.SuperchainConfigGuardianKey), *millionEth)
		l1Config.WithPrefundedAccount(addrFor(devkeys.L1ProxyAdminOwnerRole), *millionEth)
	}
}

func WithGuardianMatchL1PAO() DeployerOption {
	return func(p devtest.P, keys devkeys.Keys, builder intentbuilder.Builder) {
		_, superCfg := builder.WithSuperchain()
		intentbuilder.WithOverrideGuardianToL1PAO(p, keys, superCfg.L1ChainID(), superCfg)
	}
}

func WithPrefundedL2(l1ChainID, l2ChainID eth.ChainID) DeployerOption {
	return func(p devtest.P, keys devkeys.Keys, builder intentbuilder.Builder) {
		_, l2Config := builder.WithL2(l2ChainID)
		intentbuilder.WithDevkeyVaults(p, keys, l2Config)
		intentbuilder.WithDevkeyL2Roles(p, keys, l2Config)
		// l2configurator L1ProxyAdminOwner must be also populated
		intentbuilder.WithDevkeyL1Roles(p, keys, l2Config, l1ChainID)
		{
			faucetFunderAddr, err := keys.Address(devkeys.UserKey(funderMnemonicIndex))
			p.Require().NoError(err, "need funder addr")
			l2Config.WithPrefundedAccount(faucetFunderAddr, *eth.BillionEther.ToU256())
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

// WithInteropAtGenesis activates interop at genesis for all known L2s
func WithInteropAtGenesis() DeployerOption {
	return func(p devtest.P, keys devkeys.Keys, builder intentbuilder.Builder) {
		for _, l2Cfg := range builder.L2s() {
			l2Cfg.WithForkAtGenesis(rollup.Interop)
		}
	}
}

// WithSequencingWindow overrides the number of L1 blocks in a sequencing window, applied to all L2s.
func WithSequencingWindow(n uint64) DeployerOption {
	return func(p devtest.P, keys devkeys.Keys, builder intentbuilder.Builder) {
		builder.WithGlobalOverride("sequencerWindowSize", uint64(n))
	}
}

// WithAdditionalDisputeGames adds additional dispute games to all L2s.
func WithAdditionalDisputeGames(games []state.AdditionalDisputeGame) DeployerOption {
	return func(p devtest.P, keys devkeys.Keys, builder intentbuilder.Builder) {
		for _, l2Cfg := range builder.L2s() {
			l2Cfg.WithAdditionalDisputeGames(games)
		}
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

// WithFinalizationPeriodSeconds overrides the number of L1 blocks in a sequencing window, applied to all L2s.
func WithFinalizationPeriodSeconds(n uint64) DeployerOption {
	return func(p devtest.P, keys devkeys.Keys, builder intentbuilder.Builder) {
		for _, l2Cfg := range builder.L2s() {
			l2Cfg.WithFinalizationPeriodSeconds(n)
		}
	}
}

func WithProofMaturityDelaySeconds(n uint64) DeployerOption {
	return func(p devtest.P, keys devkeys.Keys, builder intentbuilder.Builder) {
		builder.WithGlobalOverride("proofMaturityDelaySeconds", uint64(n))
	}
}

func WithDisputeGameFinalityDelaySeconds(seconds uint64) DeployerOption {
	return func(p devtest.P, keys devkeys.Keys, builder intentbuilder.Builder) {
		builder.WithGlobalOverride("disputeGameFinalityDelaySeconds", seconds)
	}
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
		l2Genesis, l2RollupCfg, err := inspect.GenesisAndRollup(wb.output, ch.ID)
		wb.require.NoError(err, "need L2 genesis and rollup")
		id := eth.ChainIDFromBytes32(ch.ID)
		wb.outL2Genesis[id] = l2Genesis
		wb.outL2RollupCfg[id] = l2RollupCfg
	}
}

func (wb *worldBuilder) buildL2DeploymentOutputs() {
	wb.outL2Deployment = make(map[eth.ChainID]*L2Deployment)
	for _, ch := range wb.output.Chains {
		chainID := eth.ChainIDFromBytes32(ch.ID)
		wb.outL2Deployment[chainID] = &L2Deployment{
			systemConfigProxyAddr:   ch.SystemConfigProxy,
			disputeGameFactoryProxy: ch.DisputeGameFactoryProxy,
			l1StandardBridgeProxy:   ch.L1StandardBridgeProxy,
		}
	}
	wb.outSuperchainDeployment = &SuperchainDeployment{
		protocolVersionsAddr: wb.output.SuperchainDeployment.ProtocolVersionsProxy,
		superchainConfigAddr: wb.output.SuperchainDeployment.SuperchainConfigProxy,
	}
}

func (wb *worldBuilder) buildFullConfigSet() {
	// If no chain has interop active, the dep set will be nil here,
	// so we should skip building the full config set.
	if wb.output.InteropDepSet == nil {
		return
	}

	rollupConfigSet := depset.StaticRollupConfigSetFromRollupConfigMap(wb.outL2RollupCfg,
		depset.StaticTimestamp(wb.outL1Genesis.Timestamp))
	fullCfgSet, err := depset.NewFullConfigSetMerged(rollupConfigSet, wb.output.InteropDepSet)
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
	}
	for _, opt := range wb.deployerPipelineOptions {
		opt(wb, intent, &pipelineOpts)
	}

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
