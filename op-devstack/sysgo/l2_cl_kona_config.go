package sysgo

import (
	"fmt"
	"strconv"
	"time"

	coredepset "github.com/ethereum-optimism/optimism/op-core/interop/depset"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	nodeSync "github.com/ethereum-optimism/optimism/op-node/rollup/sync"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type konaNodeStartConfig struct {
	DependencySet    coredepset.DependencySet
	L2CLConfig       *L2CLConfig
	SequencerStopped bool
}

func resolveKonaL2CLConfig(t devtest.T, key string, chainID eth.ChainID, isSequencer bool, followSource string, opts []L2CLOption) *L2CLConfig {
	cfg := DefaultL2CLConfig()
	cfg.IsSequencer = isSequencer
	cfg.FollowSource = followSource
	noDiscovery, err := strconv.ParseBool(getEnvVarOrDefault("KONA_NODE_P2P_NO_DISCOVERY", "true"))
	t.Require().NoError(err, "invalid KONA_NODE_P2P_NO_DISCOVERY")
	cfg.NoDiscovery = noDiscovery
	target := NewComponentTarget(key, chainID)
	for _, opt := range opts {
		if opt != nil {
			opt.Apply(t, target, cfg)
		}
	}

	// Kona handles Engine API SYNCING responses natively; it has no switch that
	// selects op-node's CL-sync versus EL-sync behavior. Accept the recognized
	// compatibility inputs without claiming to select either native mode.
	syncMode := cfg.VerifierSyncMode
	if isSequencer {
		syncMode = cfg.SequencerSyncMode
	}
	t.Require().Contains(nodeSync.Modes, syncMode, "unknown effective sync mode for kona-node")
	t.Require().False(cfg.SafeDBPath != "" && cfg.FollowSource != "", "kona-node cannot record safe heads while delegating derivation to a follow source")

	baseCfg := *DefaultL2CLConfig()
	baseCfg.IsSequencer = isSequencer
	unsupported := *cfg
	unsupported.SequencerSyncMode = baseCfg.SequencerSyncMode
	unsupported.VerifierSyncMode = baseCfg.VerifierSyncMode
	unsupported.SafeDBPath = ""
	unsupported.VerifierL1Confs = 0
	unsupported.OffsetELSafe = 0
	unsupported.NoDiscovery = baseCfg.NoDiscovery
	unsupported.FollowSource = ""
	if isSequencer {
		unsupported.ConductorRPC = ""
	}
	// Like op-node, the timeout has no effect when no conductor URL is configured.
	unsupported.ConductorRPCTimeout = 0
	t.Require().Equal(baseCfg, unsupported, "unsupported L2CL option for kona-node")

	t.Require().GreaterOrEqual(cfg.OffsetELSafe, time.Duration(0), "EL-sync safe offset must not be negative")
	if cfg.ConductorRPC != "" {
		t.Require().Positive(cfg.ConductorRPCTimeout, "conductor RPC timeout must be positive")
		t.Require().Zero(cfg.ConductorRPCTimeout%time.Second, "Kona conductor RPC timeout must use whole seconds")
	}
	return cfg
}

func konaL2CLEnv(cfg *L2CLConfig, sequencerStopped bool) []string {
	extraEnv := []string{
		fmt.Sprintf("KONA_NODE_P2P_NO_DISCOVERY=%t", cfg.NoDiscovery),
		fmt.Sprintf("KONA_NODE_VERIFIER_L1_CONFS=%d", cfg.VerifierL1Confs),
	}
	// kona-node has no CL-sync mode: it completes EL sync whenever its execution layer has no
	// finality yet. Like op-node, apply the offset only to a node configured for EL sync. Any
	// other node keeps kona-node's 12h default, which on a young devnet places safe and
	// finalized at genesis instead of marking unsafe blocks final.
	syncMode := cfg.VerifierSyncMode
	if cfg.IsSequencer {
		syncMode = cfg.SequencerSyncMode
	}
	if syncMode == nodeSync.ELSync {
		extraEnv = append(extraEnv,
			fmt.Sprintf("KONA_NODE_SYNCMODE_OFFSET_EL_SAFE=%dms", cfg.OffsetELSafe.Milliseconds()))
	}
	if cfg.SafeDBPath != "" {
		extraEnv = append(extraEnv, "KONA_NODE_SAFEDB_PATH="+cfg.SafeDBPath)
	}
	if cfg.FollowSource != "" {
		extraEnv = append(extraEnv, "KONA_NODE_L2_FOLLOW_SOURCE="+cfg.FollowSource)
	}
	if cfg.ConductorRPC != "" {
		extraEnv = append(extraEnv,
			"KONA_NODE_CONDUCTOR_RPC="+cfg.ConductorRPC,
			fmt.Sprintf("KONA_NODE_CONDUCTOR_RPC_TIMEOUT=%d", cfg.ConductorRPCTimeout/time.Second),
		)
	}
	if sequencerStopped {
		extraEnv = append(extraEnv, "KONA_NODE_SEQUENCER_STOPPED=true")
	}
	return extraEnv
}
