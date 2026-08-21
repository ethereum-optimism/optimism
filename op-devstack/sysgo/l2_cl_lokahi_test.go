package sysgo

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-service/testreq"
)

// gateT records what a require-failure reported instead of failing the test that provoked it,
// so the preset gate can be driven for real and its message inspected.
//
// The embedded T stays nil: failLokahiUnsupportedByPresets asks only for Require, and
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
func (g *gateT) Name() string                 { return "lokahi-preset-gate" }
func (g *gateT) FailNow()                     { g.failed = true }

func (g *gateT) Errorf(format string, args ...any) {
	g.msg.WriteString(fmt.Sprintf(format, args...))
}

// The presets must turn a lokahi run away, and the reason has to name the work that would
// lift the gate. Naming the wrong blocker sends the next reader after a signature that is
// already RPC-shaped, so the three tracked gaps are asserted individually.
func TestFailLokahiUnsupportedByPresetsNamesTheBlockers(t *testing.T) {
	g := newGateT()
	failLokahiUnsupportedByPresets(g, lokahiSupernodeConfig{
		l1ELRPC:          "http://127.0.0.1:8545",
		l1BeaconAddr:     "http://127.0.0.1:5052",
		sequencerEnabled: true,
	})

	require.True(t, g.failed, "the gate must stop the run rather than let it continue")
	msg := g.msg.String()

	// The missing interop verifier, the missing query API, and the sequencing conflict.
	for _, issue := range []string{"#22537", "#22544", "#22545"} {
		require.Contains(t, msg, issue, "the gate must cite the issue tracking each blocker")
	}
	// The way out, and what the run actually asked for.
	require.Contains(t, msg, devstackSupernodeKindEnv)
	require.Contains(t, msg, string(SupernodeOpSupernode))
	require.Contains(t, msg, "sequencer=true", "the gate must report the requested configuration")

	// The previous message blamed apis.SupernodeInteropTestAPI, which #22606 made servable
	// out of process. It is no longer the blocker and must not be named as one.
	require.NotContains(t, msg, "SupernodeInteropTestAPI")
}
