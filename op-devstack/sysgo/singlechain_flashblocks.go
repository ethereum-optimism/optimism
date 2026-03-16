package sysgo

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

func startFlashblocksSingleChainPrimary(
	t devtest.T,
	keys devkeys.Keys,
	world singleChainRuntimeWorld,
	l1EL *L1Geth,
	l1CL *L1CLNode,
	jwtPath string,
	jwtSecret [32]byte,
	cfg PresetConfig,
) singleChainPrimaryRuntime {
	logger := t.Logger()

	sequencerIdentity := NewELNodeIdentity(0)
	builderIdentity := NewELNodeIdentity(0)

	l2EL := startSequencerEL(t, world.L2Network, jwtPath, jwtSecret, sequencerIdentity)
	l2Builder := startBuilderEL(t, world.L2Network, jwtPath, builderIdentity, cfg.OPRBuilderOptions...)

	connectL2ELPeers(t, logger, l2EL.UserRPC(), l2Builder.UserRPC(), false)
	connectL2ELPeers(t, logger, l2Builder.UserRPC(), l2EL.UserRPC(), true)

	rollupBoost := startRollupBoostNode(t, world.L2Network.ChainID(), l2EL, l2Builder)
	l2CL := startSequencerCL(t, keys, world.L1Network, world.L2Network, l1EL, l1CL, rollupBoost, jwtSecret, nil)

	return singleChainPrimaryRuntime{
		EL: l2EL,
		CL: l2CL,
		Flashblocks: &FlashblocksRuntimeSupport{
			Builder:     l2Builder,
			RollupBoost: rollupBoost,
		},
	}
}

func NewFlashblocksRuntime(t devtest.T) *SingleChainRuntime {
	return NewFlashblocksRuntimeWithConfig(t, PresetConfig{})
}

func NewFlashblocksRuntimeWithConfig(t devtest.T, cfg PresetConfig) *SingleChainRuntime {
	return newSingleChainRuntimeWithConfig(t, cfg, singleChainRuntimeSpec{
		BuildWorld:      newDefaultSingleChainWorld,
		StartPrimary:    startFlashblocksSingleChainPrimary,
		StartBatcher:    false,
		StartProposer:   false,
		StartChallenger: false,
	})
}

func startBuilderEL(t devtest.T, l2Net *L2Network, jwtPath string, identity *ELNodeIdentity, opts ...OPRBuilderNodeOption) *OPRBuilderNode {
	require := t.Require()

	data, err := json.Marshal(l2Net.genesis)
	require.NoError(err, "must json-encode L2 genesis")
	chainConfigPath := filepath.Join(t.TempDir(), "op-rbuilder-genesis.json")
	require.NoError(os.WriteFile(chainConfigPath, data, 0o644), "must write op-rbuilder genesis file")

	cfg := DefaultOPRbuilderNodeConfig()
	cfg.AuthRPCJWTPath = jwtPath
	cfg.Chain = chainConfigPath
	cfg.P2PAddr = "127.0.0.1"
	cfg.P2PPort = identity.Port
	cfg.P2PNodeKeyHex = identity.KeyHex()
	cfg.StaticPeers = nil
	cfg.TrustedPeers = nil
	if len(opts) > 0 {
		target := NewComponentTarget("sequencer-builder", l2Net.ChainID())
		for _, opt := range opts {
			if opt == nil {
				continue
			}
			opt.Apply(t, target, cfg)
		}
	}

	builder := &OPRBuilderNode{
		name:      "sequencer-builder",
		chainID:   l2Net.ChainID(),
		logger:    t.Logger().New("component", "op-rbuilder"),
		p:         t,
		rollupCfg: l2Net.rollupCfg,
		cfg:       cfg,
	}
	builder.Start()
	t.Cleanup(builder.Stop)
	return builder
}

func startRollupBoostNode(t devtest.T, chainID eth.ChainID, l2EL L2ELNode, builder *OPRBuilderNode) *RollupBoostNode {
	cfg := DefaultRollupBoostConfig()
	engineRPC := l2EL.EngineRPC()
	switch {
	case strings.HasPrefix(engineRPC, "ws://"):
		engineRPC = "http://" + strings.TrimPrefix(engineRPC, "ws://")
	case strings.HasPrefix(engineRPC, "wss://"):
		engineRPC = "https://" + strings.TrimPrefix(engineRPC, "wss://")
	}
	cfg.L2EngineURL = engineRPC
	cfg.L2JWTPath = l2EL.JWTPath()
	cfg.BuilderURL = ensureHTTPURL(builder.authProxyURL)
	cfg.BuilderJWTPath = builder.cfg.AuthRPCJWTPath
	cfg.FlashblocksBuilderURL = builder.wsProxyURL

	rollupBoost := &RollupBoostNode{
		name:    "rollup-boost",
		chainID: chainID,
		logger:  t.Logger().New("component", "rollup-boost"),
		p:       t,
		cfg:     cfg,
		header:  cfg.Headers,
	}
	rollupBoost.Start()
	t.Cleanup(rollupBoost.Stop)
	return rollupBoost
}
