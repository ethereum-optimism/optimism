package sysgo

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

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

// lokahiTestChain is one hosted chain with the given id and Lagoon schedule.
func lokahiTestChain(chainID uint64, lagoonTime *uint64) lokahiSupernodeChain {
	return lokahiSupernodeChain{
		net: &L2Network{
			chainID:   eth.ChainIDFromUInt64(chainID),
			rollupCfg: &rollup.Config{LagoonTime: lagoonTime},
		},
		el: &stubEL{engineRPC: fmt.Sprintf("http://127.0.0.1:9%03d", chainID%1000)},
	}
}

func u64(v uint64) *uint64 { return &v }

// The interop presets pass the activation timestamp they configured the chains' Lagoon offset
// with, so it is the number lokahi derives anyway and the run proceeds. This is the case every
// interop preset is in; a check that rejected it would gate lokahi out of all of them.
func TestLokahiAcceptsAnActivationMatchingTheRollupConfigs(t *testing.T) {
	g := newGateT()
	requireInteropActivationFromRollupConfigs(g, lokahiSupernodeConfig{
		chains: []lokahiSupernodeChain{
			lokahiTestChain(901, u64(1_700)),
			lokahiTestChain(902, u64(1_700)),
		},
		interopActivationTimestamp: u64(1_700),
	})
	require.False(t, g.failed, "a matching activation must be accepted: %s", g.msg.String())
}

// A preset that wants interop to activate somewhere other than where the chains schedule Lagoon
// is asking for something lokahi cannot do, because the message rules the verifier applies read
// that same field. Accepting it silently would run the verifier on one activation and the proof
// on another, so it is refused and both numbers are named.
func TestLokahiRejectsAnActivationTheRollupConfigsDoNotSchedule(t *testing.T) {
	g := newGateT()
	requireInteropActivationFromRollupConfigs(g, lokahiSupernodeConfig{
		chains: []lokahiSupernodeChain{
			lokahiTestChain(901, u64(1_700)),
			lokahiTestChain(902, u64(1_700)),
		},
		interopActivationTimestamp: u64(2_000),
	})

	require.True(t, g.failed, "a diverging activation must stop the run")
	msg := g.msg.String()
	require.Contains(t, msg, "1700", "the message must name what the rollup config schedules")
	require.Contains(t, msg, "2000", "the message must name what was requested")
	require.Contains(t, msg, "902", "the message must name the chain it checked")
}

// A chain that schedules no Lagoon time at all cannot activate interop, and the request names a
// timestamp, so this is a configuration that cannot be honoured rather than one where interop is
// simply off.
func TestLokahiRejectsAnActivationOnAChainWithoutLagoon(t *testing.T) {
	g := newGateT()
	requireInteropActivationFromRollupConfigs(g, lokahiSupernodeConfig{
		chains: []lokahiSupernodeChain{
			lokahiTestChain(901, u64(1_700)),
			lokahiTestChain(902, nil),
		},
		interopActivationTimestamp: u64(1_700),
	})

	require.True(t, g.failed, "a chain without Lagoon must stop the run")
	require.Contains(t, g.msg.String(), "schedules no Lagoon time")
}

// No activation requested is the non-interop case, where there is nothing to reconcile. The
// chains' own configs still decide whether lokahi runs a verifier.
func TestLokahiSkipsTheCheckWhenNoActivationIsRequested(t *testing.T) {
	g := newGateT()
	requireInteropActivationFromRollupConfigs(g, lokahiSupernodeConfig{
		chains: []lokahiSupernodeChain{lokahiTestChain(901, nil)},
	})
	require.False(t, g.failed, "no requested activation is not a conflict: %s", g.msg.String())
}
