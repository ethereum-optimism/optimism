package sysgo

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/logpipe"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
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

// TestLokahiRunningReportsAnUnaskedForExit pins what Running answers about a process that died
// on its own.
//
// This is the assertion the two-chain gate leans on to say the supernode survived losing one
// chain's execution layer. Running once read only whether a Stop had been asked for, so it
// answered true for a lokahi that had panicked and gone -- the one case the caller is asking
// about. A process that exits immediately is the cheapest stand-in for that crash: nothing on
// this side stopped it, so the old answer was true and the last check here failed.
//
// Both directions are pinned, because "always false" would satisfy the crash case on its own
// and would be a worse answer than the one it replaced.
func TestLokahiRunningReportsAnUnaskedForExit(gt *testing.T) {
	logger := testlog.Logger(gt, log.LevelInfo)
	p := devtest.NewP(context.Background(), logger,
		func(bool) { gt.Fatal("unexpected FailNow") }, func() { gt.Fatal("unexpected SkipNow") })
	gt.Cleanup(p.Close)

	discard := logpipe.LogCallback(func([]byte) {})

	// Only the fields the answer is drawn from: Running reads the process handle and nothing
	// else, so leaving the rest zero keeps the test honest about what it is exercising.
	require.False(gt, (&LokahiSupernode{}).Running(),
		"a supernode with no process is not running")

	live := NewSubProcess(p, discard, discard)
	require.NoError(gt, live.Start("/bin/sh", []string{"-c", "sleep 30"}, nil), "start the live stand-in")
	require.True(gt, (&LokahiSupernode{logger: logger, sub: live}).Running(),
		"a supernode whose process is alive must report as running")
	require.NoError(gt, live.Stop(true), "stop the live stand-in")

	gone := NewSubProcess(p, discard, discard)
	node := &LokahiSupernode{logger: logger, sub: gone}
	require.NoError(gt, gone.Start("/bin/sh", []string{"-c", "exit 3"}, nil), "start the exiting stand-in")

	// Waiting on the process rather than polling Running: Exited is closed once cmd.Wait has
	// returned, so this is the first moment the answer can be anything but true, and asking
	// then leaves no window for a pass that only means the check ran too early.
	select {
	case <-gone.Exited():
	case <-time.After(30 * time.Second):
		gt.Fatal("the stand-in process never exited")
	}

	require.False(gt, node.Running(),
		"a supernode whose process exited on its own must not report as running")
}

// TestLokahiChainRoutesShareASocketAndDifferByChain pins the addressing every caller of a
// supernode depends on: one socket, one route per chain.
//
// Both halves matter and neither implies the other. Sharing a host and port is what makes it one
// process rather than N; differing in path is what makes the two endpoints two chains. A route
// builder that dropped the chain id would still satisfy every same-host-same-port check a caller
// makes -- TestTwoChainProgress asserts exactly those -- while handing out one chain twice, so
// the collapse is asserted against directly rather than left to be inferred.
func TestLokahiChainRoutesShareASocketAndDifferByChain(gt *testing.T) {
	const supernode = "http://127.0.0.1:9545"
	chainA, chainB := eth.ChainIDFromUInt64(901), eth.ChainIDFromUInt64(902)

	routeA := chainRouteURL(supernode, chainA)
	routeB := chainRouteURL(supernode, chainB)

	uA, err := url.Parse(routeA)
	require.NoError(gt, err, "chain A's route must be a URL")
	uB, err := url.Parse(routeB)
	require.NoError(gt, err, "chain B's route must be a URL")

	require.Equal(gt, uA.Host, uB.Host, "one process serves both chains, so one host and port")
	require.Equal(gt, "/"+chainA.String(), uA.Path, "chain A is addressed by its own id")
	require.Equal(gt, "/"+chainB.String(), uB.Path, "chain B is addressed by its own id")
	require.NotEqual(gt, routeA, routeB, "two chains must never render to the same endpoint")
}
