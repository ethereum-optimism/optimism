package sysgo

import (
	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-chain-ops/interopgen/config"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-core/forks"
	coredepset "github.com/ethereum-optimism/optimism/op-core/interop/depset"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/intentbuilder"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/ptr"
)

func newWorldBuilder(t devtest.T, keys devkeys.Keys) *worldBuilder {
	return &worldBuilder{
		p:       t,
		logger:  t.Logger(),
		require: t.Require(),
		keys:    keys,
		builder: intentbuilder.New(),
	}
}

func applyConfigInteropAtGenesis(builder intentbuilder.Builder) {
	for _, l2Cfg := range builder.L2s() {
		l2Cfg.WithForkAtGenesis(forks.Lagoon)
	}
}

func applyConfigDeployerOptions(t devtest.T, keys devkeys.Keys, builder intentbuilder.Builder, opts []DeployerOption) {
	if len(opts) == 0 {
		return
	}
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		opt(t, keys, builder)
	}
}

func genesisAnchorGameType(cfg PresetConfig) *uint32 {
	var result *uint32
	for _, games := range [][]gameTypes.GameType{cfg.AddedGameTypes, cfg.RespectedGameTypes} {
		for _, game := range games {
			switch game {
			case gameTypes.SuperCannonKonaGameType, gameTypes.ZKDisputeGameType:
				return ptr.New(uint32(gameTypes.SuperCannonKonaGameType))
			case gameTypes.CannonKonaGameType:
				result = ptr.New(uint32(game))
			}
		}
	}
	return result
}

type interopMigrationState struct {
	opcmImpl             common.Address
	superchainConfigAddr common.Address
	l2Deployments        map[eth.ChainID]*L2Deployment
}

func newInteropMigrationState(wb *worldBuilder) *interopMigrationState {
	if wb == nil || wb.output == nil || wb.outSuperchainDeployment == nil {
		return nil
	}
	state := &interopMigrationState{
		opcmImpl:             wb.output.ImplementationsDeployment.OpcmV2Impl,
		superchainConfigAddr: wb.outSuperchainDeployment.SuperchainConfigAddr(),
		l2Deployments:        make(map[eth.ChainID]*L2Deployment, len(wb.outL2Deployment)),
	}
	for chainID, deployment := range wb.outL2Deployment {
		state.l2Deployments[chainID] = deployment
	}
	return state
}

func buildSingleChainWorld(t devtest.T, keys devkeys.Keys, lagoonAtGenesis bool, localContractArtifactsPath string, anchorGameType *uint32, deployerOpts ...DeployerOption) (*interopMigrationState, *L1Network, *L2Network, coredepset.DependencySet, config.FullConfigSetMerged) {
	wb := newWorldBuilder(t, keys)
	wb.genesisAnchorGameType = anchorGameType
	applyConfigLocalContractSources(t, keys, wb.builder, localContractArtifactsPath)
	applyConfigCommons(t, keys, DefaultL1ID, wb.builder)
	applyConfigPrefundedL2(t, keys, DefaultL1ID, DefaultL2AID, wb.builder)
	if lagoonAtGenesis {
		applyConfigInteropAtGenesis(wb.builder)
	}
	applyConfigDeployerOptions(t, keys, wb.builder, deployerOpts)
	wb.Build()

	t.Require().Len(wb.l2Chains, 1, "expected exactly one L2 chain")
	l2ID := wb.l2Chains[0]
	l1ID := eth.ChainIDFromUInt64(wb.output.AppliedIntent.L1ChainID)

	l1Net := &L1Network{
		name:         "l1",
		chainID:      l1ID,
		genesis:      wb.outL1Genesis,
		blockTime:    6,
		deployChains: wb.deployChains,
	}
	l2Net := &L2Network{
		name:       "l2a",
		chainID:    l2ID,
		l1ChainID:  l1ID,
		genesis:    wb.outL2Genesis[l2ID],
		rollupCfg:  wb.outL2RollupCfg[l2ID],
		deployment: wb.outL2Deployment[l2ID],
		opcmImpl:   wb.output.ImplementationsDeployment.OpcmV2Impl,
		mipsImpl:   wb.output.ImplementationsDeployment.MipsImpl,
		keys:       keys,
	}
	var depSet coredepset.DependencySet
	if wb.outFullCfgSet.DependencySet != nil {
		depSet = wb.outFullCfgSet.DependencySet
	}
	return newInteropMigrationState(wb), l1Net, l2Net, depSet, wb.outFullCfgSet
}
