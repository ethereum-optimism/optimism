// Temporary harness: a single-chain flashblocks runtime backed by one
// op-reth-premium process (sequencer EL + in-process subblocks producer), with
// no rollup-boost and no separate builder EL. Not intended to merge.
package sysgo

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
)

// PremiumNodeOption mutates a PremiumNodeConfig before launch.
type PremiumNodeOption func(p devtest.CommonT, target ComponentTarget, cfg *PremiumNodeConfig)

func startPremiumSingleChainPrimary(
	t devtest.T,
	keys devkeys.Keys,
	world singleChainRuntimeWorld,
	l1EL *L1Geth,
	l1CL *L1CLNode,
	jwtPath string,
	jwtSecret [32]byte,
	cfg PresetConfig,
) singleChainPrimaryRuntime {
	identity := NewELNodeIdentity(0)
	premium := startPremiumNode(t, world.L2Network, jwtPath, identity, cfg.PremiumOptions...)

	var l2CL L2CLNode
	if world.Interop != nil {
		l2CL = startL2CLNode(t, keys, world.L1Network, world.L2Network, l1EL, l1CL, premium, jwtSecret, l2CLNodeStartConfig{
			Key:           "sequencer",
			IsSequencer:   true,
			NoDiscovery:   true,
			EnableReqResp: true,
			UseReqResp:    true,
			DependencySet: world.Interop.DependencySet,
		})
	} else {
		l2CL = startSequencerCL(t, keys, world.L1Network, world.L2Network, l1EL, l1CL, premium, jwtSecret, nil)
	}

	return singleChainPrimaryRuntime{
		EL: premium,
		CL: l2CL,
		Flashblocks: &FlashblocksRuntimeSupport{
			Builder:     premium,
			RollupBoost: premium,
		},
	}
}

// NewPremiumFlashblocksRuntime builds a single-chain flashblocks runtime backed
// by op-reth-premium.
func NewPremiumFlashblocksRuntime(t devtest.T) *SingleChainRuntime {
	return NewPremiumFlashblocksRuntimeWithConfig(t, PresetConfig{})
}

func NewPremiumFlashblocksRuntimeWithConfig(t devtest.T, cfg PresetConfig) *SingleChainRuntime {
	return newSingleChainRuntimeWithConfig(t, cfg, singleChainRuntimeSpec{
		BuildWorld:      newDefaultSingleChainWorld,
		StartPrimary:    startPremiumSingleChainPrimary,
		StartBatcher:    false,
		StartProposer:   false,
		StartChallenger: false,
	})
}

func startPremiumNode(t devtest.T, l2Net *L2Network, jwtPath string, identity *ELNodeIdentity, opts ...PremiumNodeOption) *PremiumNode {
	require := t.Require()

	data, err := json.Marshal(l2Net.genesis)
	require.NoError(err, "must json-encode L2 genesis")
	chainConfigPath := filepath.Join(t.TempDir(), "op-reth-premium-genesis.json")
	require.NoError(os.WriteFile(chainConfigPath, data, 0o644), "must write op-reth-premium genesis file")

	cfg := DefaultPremiumNodeConfig()
	cfg.ChainBlockTime = time.Second * time.Duration(l2Net.rollupCfg.BlockTime)
	cfg.AuthRPCJWTPath = jwtPath
	cfg.Chain = chainConfigPath
	cfg.P2PAddr = "127.0.0.1"
	cfg.P2PPort = identity.Port
	cfg.P2PNodeKeyHex = identity.KeyHex()

	if len(opts) > 0 {
		target := NewComponentTarget("sequencer-premium", l2Net.ChainID())
		for _, opt := range opts {
			if opt == nil {
				continue
			}
			opt(t, target, cfg)
		}
	}

	premium := &PremiumNode{
		name:      "sequencer-premium",
		chainID:   l2Net.ChainID(),
		logger:    t.Logger().New("component", "op-reth-premium"),
		p:         t,
		rollupCfg: l2Net.rollupCfg,
		cfg:       cfg,
	}
	premium.Start()
	t.Cleanup(premium.Stop)
	return premium
}
