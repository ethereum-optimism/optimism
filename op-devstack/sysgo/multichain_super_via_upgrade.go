package sysgo

import (
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
)

// attachSupernodeSuperProofsViaUpgrade is the opcm.upgrade counterpart to
// attachSupernodeSuperProofs. It installs the super-root dispute games through
// the standard OPCMv2 upgrade entrypoint, then wires the interop challenger and
// super proposer like the migration-based helper does.
func attachSupernodeSuperProofsViaUpgrade(t devtest.T, runtime *MultiChainRuntime, cfg PresetConfig) *MultiChainRuntime {
	chains := orderedRuntimeChains(runtime)
	t.Require().NotEmpty(chains, "supernode superproofs runtime must contain at least one chain")
	t.Require().NotNil(runtime.Supernode, "supernode superproofs runtime must provide a supernode")

	proofChain := chains[0]
	cls := make([]L2CLNode, 0, len(chains))
	for _, chain := range chains {
		t.Require().NotNil(chain, "runtime chain entry must not be nil")
		cls = append(cls, chain.CL)
	}

	superrootTime := awaitSuperrootTime(t, cls...)
	superRoot := getSupernodeSuperRoot(t, runtime.Supernode, superrootTime)
	upgradeToSuperRoots(t, runtime.Keys, runtime.Migration, runtime.L1Network.ChainID(), runtime.L1EL, superRoot, superrootTime, proofChain.Network.ChainID())

	attachSuperChallengerAndProposer(t, runtime, cfg, gameTypes.SuperCannonGameType)
	return runtime
}

// NewSingleChainSupernodeProofsUpgradeRuntimeWithConfig deploys a single-chain
// interop runtime and switches it over to super-root dispute games via a call to
// opcm.upgrade. The SUPER_ROOT_GAMES_MIGRATION dev feature is enabled on the
// OPCM so that the output-root → super-root switch is permitted during the
// upgrade (including the startingAnchorRoot override). Unlike
// opcmMigrator.migrate — which is specifically about moving multiple chains
// onto a shared DisputeGameFactory — opcm.upgrade is the standard per-chain
// upgrade entrypoint, which is what single-chain tests want.
func NewSingleChainSupernodeProofsUpgradeRuntimeWithConfig(t devtest.T, interopAtGenesis bool, cfg PresetConfig) *MultiChainRuntime {
	cfg = withSuperRootGamesAtGenesisDeployerFeatures(cfg)
	runtime := newSingleChainSupernodeRuntimeWithConfig(t, interopAtGenesis, cfg)
	// No minimal output-root proposer: with SUPER_ROOT_GAMES_MIGRATION enabled
	// the chain has no PermissionedCannon game at genesis, so a legacy proposer
	// would just loop on NoImplementation reverts. The super proposer is
	// attached by attachSupernodeSuperProofsViaUpgrade.
	attachTestSequencerToRuntime(t, runtime, "dev")
	return attachSupernodeSuperProofsViaUpgrade(t, runtime, cfg)
}
