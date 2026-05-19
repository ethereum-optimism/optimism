package sysgo

import (
	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-core/forks"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/intentbuilder"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/depset"
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
		l2Cfg.WithForkAtGenesis(forks.Interop)
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

func buildSingleChainWorldWithInterop(t devtest.T, keys devkeys.Keys, interopAtGenesis bool, localContractArtifactsPath string, deployerOpts ...DeployerOption) (*L1Network, *L2Network, depset.DependencySet, depset.FullConfigSetMerged) {
	_, l1Net, l2Net, depSet, fullCfgSet := buildSingleChainWorldWithInteropAndState(t, keys, interopAtGenesis, localContractArtifactsPath, deployerOpts...)
	return l1Net, l2Net, depSet, fullCfgSet
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

func buildSingleChainWorldWithInteropAndState(t devtest.T, keys devkeys.Keys, interopAtGenesis bool, localContractArtifactsPath string, deployerOpts ...DeployerOption) (*interopMigrationState, *L1Network, *L2Network, depset.DependencySet, depset.FullConfigSetMerged) {
	wb := newWorldBuilder(t, keys)
	applyConfigLocalContractSources(t, keys, wb.builder, localContractArtifactsPath)
	applyConfigCommons(t, keys, DefaultL1ID, wb.builder)
	applyConfigPrefundedL2(t, keys, DefaultL1ID, DefaultL2AID, wb.builder)
	if interopAtGenesis {
		applyConfigInteropAtGenesis(wb.builder)
	}
	applyConfigDeployerOptions(t, keys, wb.builder, deployerOpts)
	wb.Build()

	t.Require().Len(wb.l2Chains, 1, "expected exactly one L2 chain")
	l2ID := wb.l2Chains[0]
	l1ID := eth.ChainIDFromUInt64(wb.output.AppliedIntent.L1ChainID)

	l1Net := &L1Network{
		name:      "l1",
		chainID:   l1ID,
		genesis:   wb.outL1Genesis,
		blockTime: 6,
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
	var depSet depset.DependencySet
	if wb.outFullCfgSet.DependencySet != nil {
		depSet = wb.outFullCfgSet.DependencySet
	}
	return newInteropMigrationState(wb), l1Net, l2Net, depSet, wb.outFullCfgSet
}
