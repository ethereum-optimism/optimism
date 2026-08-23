package sysgo

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/params"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testreq"
)

// gateT records what a require-failure reported instead of failing the test that provoked it,
// so a fail-fast can be driven for real and its message inspected.
//
// The embedded T stays nil: the checks under test ask only for Require, and
// lokahiSupernodeConfig.describe needs no test scope. Name is overridden too because testify
// reaches for it while rendering a failure, and would otherwise reach it through that nil.
type gateT struct {
	devtest.T
	msg    strings.Builder
	failed bool
}

func newGateT() *gateT { return &gateT{} }

func (g *gateT) Require() *testreq.Assertions { return testreq.New(g) }
func (g *gateT) Helper()                      {}
func (g *gateT) Name() string                 { return "lokahi-config-check" }
func (g *gateT) FailNow()                     { g.failed = true }

func (g *gateT) Errorf(format string, args ...any) {
	g.msg.WriteString(fmt.Sprintf(format, args...))
}

// stubEL stands in for a chain's execution layer where only its address is read.
// lokahiSupernodeConfig.describe renders EngineRPC while building a failure message, so a
// config driven through a check that fails needs one.
type stubEL struct{ engineRPC string }

var _ L2ELNode = (*stubEL)(nil)

func (e *stubEL) Start()            {}
func (e *stubEL) Stop()             {}
func (e *stubEL) UserRPC() string   { return e.engineRPC }
func (e *stubEL) EngineRPC() string { return e.engineRPC }
func (e *stubEL) JWTPath() string   { return "/dev/null" }

func u64(v uint64) *uint64 { return &v }

// The activation timestamp is no longer refused, it is passed on: lokahi takes an [interop]
// activation-timestamp and reconciles it with each chain's Lagoon time itself, in the one place
// that sees the whole hosted set. What the devstack owes it is the number the preset computed,
// spelled the way the config file expects.
func TestLokahiConfigCarriesTheRequestedInteropActivation(t *testing.T) {
	const activation = 1_787_335_282
	cfg := lokahiSupernodeConfig{
		l1Net:                      &L1Network{genesis: &core.Genesis{Config: params.MainnetChainConfig}},
		l1ELRPC:                    "http://127.0.0.1:8545",
		l1BeaconAddr:               "http://127.0.0.1:5052",
		interopActivationTimestamp: u64(activation),
	}
	rendered := lokahiConfigFile(newGateT(), t.TempDir(), cfg, nil)

	require.Contains(t, rendered, "[interop]", "an interop table must be written")
	require.Contains(t, rendered, fmt.Sprintf("activation-timestamp = %d", activation),
		"the requested activation must reach lokahi verbatim")
}

// Every hosted chain serves the experimental opstack block-building namespace, because
// op-supernode's virtual op-nodes do (makeNodeCfg sets ExperimentalOPStackAPI on all of them):
// the test sequencer drives block building through opstack_* on each chain's route.
func TestLokahiConfigEnablesTheOpstackNamespace(t *testing.T) {
	cfg := lokahiSupernodeConfig{
		l1Net:        &L1Network{genesis: &core.Genesis{Config: params.MainnetChainConfig}},
		l1ELRPC:      "http://127.0.0.1:8545",
		l1BeaconAddr: "http://127.0.0.1:5052",
	}
	rendered := lokahiConfigFile(newGateT(), t.TempDir(), cfg, nil)

	require.Contains(t, rendered, "experimental-opstack-api = true",
		"the devstack must turn the opstack namespace on, as it does on op-supernode")
}

// A chain's P2P listeners ask the kernel for ephemeral ports rather than being handed a port the
// harness picked ahead of time: a reserved port is free again before lokahi binds it, and under
// parallel tests another component can take it in that window — the discovery service dies on
// the collision and takes the whole node with it. Port 0 is how startMixedKonaNode runs
// kona-node and how newDevstackP2PConfig runs op-node, so lokahi must be launched the same way.
func TestLokahiChainEntryBindsEphemeralP2PPorts(t *testing.T) {
	keys, err := devkeys.NewMnemonicDevKeys(devkeys.TestMnemonic)
	require.NoError(t, err)
	chain := lokahiSupernodeChain{
		net: &L2Network{
			chainID:   eth.ChainIDFromUInt64(901),
			rollupCfg: &rollup.Config{},
			keys:      keys,
		},
		el: &stubEL{engineRPC: "http://127.0.0.1:9551"},
	}

	entry := lokahiChainEntry(newGateT(), t.TempDir(), chain, lokahiSupernodeConfig{})

	require.Contains(t, entry, "p2p-tcp-port = 0", "gossip must bind an ephemeral port")
	require.Contains(t, entry, "p2p-udp-port = 0", "discovery must bind an ephemeral port")
}

// A preset that requests no activation must not write the table at all, so a node that was not
// told one keeps reading its activation from the rollup configs -- the default path, unchanged.
func TestLokahiConfigOmitsInteropWhenNoActivationIsRequested(t *testing.T) {
	cfg := lokahiSupernodeConfig{
		l1Net:        &L1Network{genesis: &core.Genesis{Config: params.MainnetChainConfig}},
		l1ELRPC:      "http://127.0.0.1:8545",
		l1BeaconAddr: "http://127.0.0.1:5052",
	}
	rendered := lokahiConfigFile(newGateT(), t.TempDir(), cfg, nil)

	require.NotContains(t, rendered, "[interop]",
		"no requested activation must leave lokahi reading the forks")
	require.NotContains(t, rendered, "activation-timestamp")
}
