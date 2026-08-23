package sysgo

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/params"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
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

// The devnet L1 finalizes within seconds, and both SuperRootMigrator tests budget 150s for the
// supernode's first finalized advance. op-node sees it in time because the devstack hands it a
// 2-second L1EpochPollInterval; lokahi has to be handed the same cadence, or kona's 60-second
// default leaves first finality (~126s in) unobserved until the ~180s poll.
func TestLokahiConfigMatchesOpNodesEpochPollInterval(t *testing.T) {
	cfg := lokahiSupernodeConfig{
		l1Net:        &L1Network{genesis: &core.Genesis{Config: params.MainnetChainConfig}},
		l1ELRPC:      "http://127.0.0.1:8545",
		l1BeaconAddr: "http://127.0.0.1:5052",
	}
	rendered := lokahiConfigFile(newGateT(), t.TempDir(), cfg, nil)

	require.Contains(t, rendered, fmt.Sprintf("epoch-poll-interval = %d", lokahiL1EpochPollSeconds),
		"lokahi must poll L1 finality on the cadence the devstack hands op-node")
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
