package sysgo

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	nodeSync "github.com/ethereum-optimism/optimism/op-node/rollup/sync"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testreq"
)

func TestKonaL2CLEnvSupportedOptions(t *testing.T) {
	t.Setenv("KONA_NODE_P2P_NO_DISCOVERY", "true")
	chainID := eth.ChainIDFromUInt64(901)
	for _, isSequencer := range []bool{false, true} {
		t.Run(fmt.Sprintf("sequencer=%t", isSequencer), func(t *testing.T) {
			dt := devtest.SerialT(t)
			opts := []L2CLOption{
				L2CLOptionFn(func(_ devtest.T, target ComponentTarget, cfg *L2CLConfig) {
					require.Equal(t, NewComponentTarget("node", chainID), target)
					require.Equal(t, isSequencer, cfg.IsSequencer)
					cfg.SafeDBPath = "/tmp/safe-head.db"
					cfg.NoDiscovery = false
				}),
			}
			require.Equal(t, []string{
				"KONA_NODE_P2P_NO_DISCOVERY=false",
				"KONA_NODE_VERIFIER_L1_CONFS=0",
				"KONA_NODE_SAFEDB_PATH=/tmp/safe-head.db",
			}, konaL2CLEnv(resolveKonaL2CLConfig(dt, "node", chainID, isSequencer, "", opts), false))
		})
	}
}

func TestKonaL2CLEnvDiscovery(t *testing.T) {
	for _, ambient := range []string{"true", "false"} {
		t.Run(ambient, func(t *testing.T) {
			t.Setenv("KONA_NODE_P2P_NO_DISCOVERY", ambient)
			dt := devtest.SerialT(t)
			chainID := eth.ChainIDFromUInt64(901)
			require.Equal(t, []string{"KONA_NODE_P2P_NO_DISCOVERY=" + ambient, "KONA_NODE_VERIFIER_L1_CONFS=0"},
				konaL2CLEnv(resolveKonaL2CLConfig(dt, "node", chainID, false, "", nil), false))
			for _, noDiscovery := range []bool{false, true} {
				opts := []L2CLOption{L2CLOptionFn(func(_ devtest.T, _ ComponentTarget, cfg *L2CLConfig) {
					cfg.NoDiscovery = noDiscovery
				})}
				require.Equal(t, []string{fmt.Sprintf("KONA_NODE_P2P_NO_DISCOVERY=%t", noDiscovery), "KONA_NODE_VERIFIER_L1_CONFS=0"},
					konaL2CLEnv(resolveKonaL2CLConfig(dt, "node", chainID, false, "", opts), false))
			}
		})
	}
}

func TestKonaL2CLEnvFollowSource(t *testing.T) {
	t.Setenv("KONA_NODE_P2P_NO_DISCOVERY", "true")
	dt := devtest.SerialT(t)
	chainID := eth.ChainIDFromUInt64(901)
	require.Equal(t, []string{
		"KONA_NODE_P2P_NO_DISCOVERY=true",
		"KONA_NODE_VERIFIER_L1_CONFS=0",
		"KONA_NODE_L2_FOLLOW_SOURCE=http://initial:8545",
	}, konaL2CLEnv(resolveKonaL2CLConfig(dt, "node", chainID, false, "http://initial:8545", nil), false))
	// Apply callbacks in order, with the same initial follow source as op-node.
	require.Equal(t, []string{
		"KONA_NODE_P2P_NO_DISCOVERY=true",
		"KONA_NODE_VERIFIER_L1_CONFS=0",
		"KONA_NODE_L2_FOLLOW_SOURCE=http://last:8545",
	}, konaL2CLEnv(resolveKonaL2CLConfig(dt, "node", chainID, false, "http://initial:8545", []L2CLOption{
		L2CLOptionFn(func(_ devtest.T, _ ComponentTarget, config *L2CLConfig) {
			require.Equal(t, "http://initial:8545", config.FollowSource)
			config.FollowSource = "http://first:8545"
		}),
		L2CLFollowSource("http://last:8545"),
	}), false))
}

func TestKonaL2CLEnvAcceptsSyncModeCompatibilityInputs(t *testing.T) {
	t.Setenv("KONA_NODE_P2P_NO_DISCOVERY", "true")
	for _, isSequencer := range []bool{false, true} {
		for _, mode := range nodeSync.Modes {
			t.Run(fmt.Sprintf("sequencer=%t/mode=%s", isSequencer, mode), func(t *testing.T) {
				opts := []L2CLOption{L2CLOptionFn(func(_ devtest.T, _ ComponentTarget, cfg *L2CLConfig) {
					// Only the effective role's sync mode matters, just like op-node.
					cfg.SequencerSyncMode, cfg.VerifierSyncMode = nodeSync.Mode(99), nodeSync.Mode(99)
					if isSequencer {
						cfg.SequencerSyncMode = mode
					} else {
						cfg.VerifierSyncMode = mode
					}
				})}
				// Neither compatibility mode emits a flag pretending to select a
				// native mode: Kona always drives EL sync through Engine API responses.
				require.Equal(t, []string{"KONA_NODE_P2P_NO_DISCOVERY=true", "KONA_NODE_VERIFIER_L1_CONFS=0"},
					konaL2CLEnv(resolveKonaL2CLConfig(devtest.SerialT(t), "node", eth.ChainIDFromUInt64(901), isSequencer, "", opts), false))
			})
		}
	}
}

func TestKonaL2CLEnvConductor(t *testing.T) {
	t.Setenv("KONA_NODE_P2P_NO_DISCOVERY", "true")
	opts := []L2CLOption{L2CLOptionFn(func(_ devtest.T, _ ComponentTarget, cfg *L2CLConfig) {
		cfg.ConductorRPC = "http://conductor:8547"
		cfg.ConductorRPCTimeout = 7 * time.Second
	})}
	require.Equal(t, []string{
		"KONA_NODE_P2P_NO_DISCOVERY=true",
		"KONA_NODE_VERIFIER_L1_CONFS=0",
		"KONA_NODE_CONDUCTOR_RPC=http://conductor:8547",
		"KONA_NODE_CONDUCTOR_RPC_TIMEOUT=7",
	}, konaL2CLEnv(resolveKonaL2CLConfig(devtest.SerialT(t), "node", eth.ChainIDFromUInt64(901), true, "", opts), false))
}

// Capture setup assertions without failing the enclosing regression test.
type rejectedKonaOptionTest struct {
	devtest.T
	failure string
}

func (t *rejectedKonaOptionTest) Require() *testreq.Assertions { return testreq.New(t) }

func (t *rejectedKonaOptionTest) Errorf(format string, args ...any) {
	t.failure = fmt.Sprintf(format, args...)
	panic("Kona option rejected")
}

func TestKonaL2CLConfigRejectsUnsupportedOptions(t *testing.T) {
	t.Setenv("KONA_NODE_P2P_NO_DISCOVERY", "true")
	for _, tc := range []struct {
		name        string
		isSequencer bool
		apply       func(*L2CLConfig)
		message     string
	}{
		{"safe offset", true, func(cfg *L2CLConfig) { cfg.OffsetELSafe = time.Second }, "unsupported L2CL option"},
		{"role override", true, func(cfg *L2CLConfig) { cfg.IsSequencer = false }, "unsupported L2CL option"},
		{"sequencer sync", true, func(cfg *L2CLConfig) { cfg.SequencerSyncMode = nodeSync.Mode(99) }, "unknown effective sync mode"},
		{"verifier sync", false, func(cfg *L2CLConfig) { cfg.VerifierSyncMode = nodeSync.Mode(99) }, "unknown effective sync mode"},
		{"verifier conductor", false, func(cfg *L2CLConfig) { cfg.ConductorRPC = "http://conductor:8547" }, "unsupported L2CL option"},
		{"zero conductor timeout", true, func(cfg *L2CLConfig) { cfg.ConductorRPC = "http://conductor:8547" }, "timeout must be positive"},
		{"fractional conductor timeout", true, func(cfg *L2CLConfig) {
			cfg.ConductorRPC, cfg.ConductorRPCTimeout = "http://conductor:8547", time.Millisecond
		}, "must use whole seconds"},
		{"delegated safe database", false, func(cfg *L2CLConfig) {
			cfg.SafeDBPath, cfg.FollowSource = "safe-head.db", "http://source:8545"
		}, "cannot record safe heads"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dt := &rejectedKonaOptionTest{T: devtest.SerialT(t)}
			opts := []L2CLOption{L2CLOptionFn(func(_ devtest.T, _ ComponentTarget, cfg *L2CLConfig) { tc.apply(cfg) })}
			require.PanicsWithValue(t, "Kona option rejected", func() {
				resolveKonaL2CLConfig(dt, "node", eth.ChainIDFromUInt64(901), tc.isSequencer, "", opts)
			})
			require.Contains(t, dt.failure, tc.message)
		})
	}
}

func TestKonaL2CLEnvSequencerStopped(t *testing.T) {
	for _, testCase := range []struct {
		environment []string
		name        string
		stopped     bool
	}{
		{environment: []string{"KONA_NODE_P2P_NO_DISCOVERY=false", "KONA_NODE_VERIFIER_L1_CONFS=0"}, name: "running"},
		{environment: []string{"KONA_NODE_P2P_NO_DISCOVERY=false", "KONA_NODE_VERIFIER_L1_CONFS=0", "KONA_NODE_SEQUENCER_STOPPED=true"}, name: "stopped", stopped: true},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			cfg := DefaultL2CLConfig()
			cfg.IsSequencer = true
			require.Equal(t, testCase.environment, konaL2CLEnv(cfg, testCase.stopped))
		})
	}
}

func TestResolveKonaL2CLConfigSafeDBOptions(t *testing.T) {
	t.Setenv("KONA_NODE_P2P_NO_DISCOVERY", "true")
	safeDBOption := func(path string) L2CLOption {
		return L2CLOptionFn(func(_ devtest.T, _ ComponentTarget, config *L2CLConfig) {
			config.SafeDBPath = path
		})
	}
	for _, testCase := range []struct {
		name     string
		options  []L2CLOption
		wantPath string
	}{
		{name: "disabled by default"},
		{name: "enabled", options: []L2CLOption{safeDBOption("/tmp/safe-head.db")}, wantPath: "/tmp/safe-head.db"},
		{name: "nil option ignored", options: []L2CLOption{nil, safeDBOption("/tmp/safe-head.db")}, wantPath: "/tmp/safe-head.db"},
		{name: "later option replaces path", options: []L2CLOption{safeDBOption("/tmp/first.db"), safeDBOption("/tmp/second.db")}, wantPath: "/tmp/second.db"},
		{name: "later option disables database", options: []L2CLOption{safeDBOption("/tmp/safe-head.db"), safeDBOption("")}},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			config := resolveKonaL2CLConfig(devtest.SerialT(t), "node", eth.ChainIDFromUInt64(901), false, "", testCase.options)

			require.Equal(t, testCase.wantPath, config.SafeDBPath)
		})
	}
}

func TestKonaL2CLEnvVerifierL1Confs(t *testing.T) {
	for _, isSequencer := range []bool{false, true} {
		for _, confirmations := range []uint64{0, 4} {
			t.Run(fmt.Sprintf("sequencer=%t/confs=%d", isSequencer, confirmations), func(t *testing.T) {
				opts := []L2CLOption{L2CLOptionFn(func(_ devtest.T, _ ComponentTarget, cfg *L2CLConfig) {
					cfg.VerifierL1Confs = confirmations
				})}
				cfg := resolveKonaL2CLConfig(devtest.SerialT(t), "node", eth.ChainIDFromUInt64(901), isSequencer, "", opts)
				require.Contains(t, konaL2CLEnv(cfg, false), fmt.Sprintf("KONA_NODE_VERIFIER_L1_CONFS=%d", confirmations))
			})
		}
	}
}
