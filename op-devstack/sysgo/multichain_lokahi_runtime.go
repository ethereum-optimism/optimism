package sysgo

import (
	"encoding/json"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-service/clock"
)

// NewTwoL2LokahiRuntime starts two chains with interop active at genesis, each sequenced and
// derived by its own kona-node and op-reth, and one lokahi process verifying both.
//
// Each kona-node derives local-safe from L1 by itself, records it in a safe head database, and
// follows lokahi for its safe and finalized labels over JSON-RPC. lokahi reads both chains'
// local-safe blocks from those same nodes. No supernode runs.
func NewTwoL2LokahiRuntime(t devtest.T) *MultiChainRuntime {
	require := t.Require()

	keys, err := devkeys.NewMnemonicDevKeys(devkeys.TestMnemonic)
	require.NoError(err, "failed to derive dev keys from mnemonic")

	jwtPath, jwtSecret := writeJWTSecret(t)
	var l1EL *L1Geth
	var l1CL *L1CLNode
	chainSpecs := []runtimeChainSpec{
		{Name: "l2a", ID: DefaultL2AID},
		{Name: "l2b", ID: DefaultL2BID},
	}
	wb, l1Net, l2Nets := buildMultiL2RuntimeWorld(
		t, keys, true, 0, "", chainSpecs, nil,
		func(l1Net *L1Network) (*L1Geth, *L1CLNode) {
			l1EL, l1CL = startInProcessL1WithClockConfig(t, l1Net, jwtPath, clock.SystemClock, PresetConfig{})
			return l1EL, l1CL
		},
	)
	depSet := wb.output.InteropDepSet
	require.NotNil(depSet, "op-deployer must generate the interop dependency set")
	depSetJSON, err := json.Marshal(depSet)
	require.NoError(err, "marshal interop dependency set")

	lokahi := newLokahi(t)

	chains := make(map[string]*MultiChainNodeRuntime, len(chainSpecs))
	lokahiChains := make([]LokahiChain, 0, len(chainSpecs))
	for i, chainSpec := range chainSpecs {
		l2Net := l2Nets[i]
		el := startSequencerEL(t, l2Net, jwtPath, jwtSecret)
		safeDBPath := t.TempDirWithPrefix("safedb-" + NewComponentTarget("sequencer", l2Net.ChainID()).String())
		cl := startMixedKonaNode(t, keys, l1Net, l2Net, l1EL, l1CL, el, "sequencer", "sequencer", konaNodeStartConfig{
			DependencySet: depSet,
			L2CLConfig: resolveKonaL2CLConfig(t, "sequencer", l2Net.ChainID(), true, "", []L2CLOption{
				L2CLOptionFn(func(_ devtest.T, _ ComponentTarget, cfg *L2CLConfig) {
					cfg.SafeDBPath = safeDBPath
				}),
			}),
			InteropAuthorityRPC: lokahi.UserRPC(),
		})
		batcher := startMinimalBatcher(t, keys, l2Net, l1EL, cl, el)
		chains[chainSpec.Name] = &MultiChainNodeRuntime{
			Name:    chainSpec.Name,
			Network: l2Net,
			EL:      el,
			CL:      cl,
			Batcher: batcher,
		}
		lokahiChains = append(lokahiChains, LokahiChain{Network: l2Net, SourceCL: cl, SourceEL: el})
	}

	lokahi.configure(l1EL, depSetJSON, lokahiChains)
	lokahi.Start()
	t.Cleanup(lokahi.Stop)

	return &MultiChainRuntime{
		Keys:          keys,
		Migration:     newInteropMigrationState(wb),
		DependencySet: depSet,
		L1Network:     l1Net,
		L1EL:          l1EL,
		L1CL:          l1CL,
		Chains:        chains,
		Lokahi:        lokahi,
	}
}
