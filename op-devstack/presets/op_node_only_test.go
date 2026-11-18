package presets

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/stretchr/testify/require"
)

type presetCase struct {
	name      string
	newPreset func(devtest.T)
}

// These presets start only op-node and op-supernode CLs, so under kona-node each must skip before
// it starts anything.
func TestOpNodeOnlyPresetsSkipOnKonaNode(t *testing.T) {
	t.Setenv("DEVSTACK_L2CL_KIND", string(sysgo.MixedL2CLKona))
	for _, p := range []presetCase{
		{"NewSimpleInterop", func(t devtest.T) { NewSimpleInterop(t) }},
		{"NewSimpleInteropIsthmusSuper", func(t devtest.T) { NewSimpleInteropIsthmusSuper(t) }},
		{"NewThreeChainInterop", func(t devtest.T) { NewThreeChainInterop(t) }},
		{"NewSingleChainInterop", func(t devtest.T) { NewSingleChainInterop(t) }},
		{"NewSingleChainInteropIsthmusSuper", func(t devtest.T) { NewSingleChainInteropIsthmusSuper(t) }},
		{"NewSingleChainInteropSuperRootAtGenesis", func(t devtest.T) { NewSingleChainInteropSuperRootAtGenesis(t) }},
		{"NewSingleChainInteropNoSupernode", func(t devtest.T) { NewSingleChainInteropNoSupernode(t) }},
		{"NewSingleChainInteropNoSupernodeZKDispute", func(t devtest.T) { NewSingleChainInteropNoSupernodeZKDispute(t) }},
		{"NewMinimalInteropNoSupernode", func(t devtest.T) { NewMinimalInteropNoSupernode(t) }},
		{"NewTwoL2Supernode", func(t devtest.T) { NewTwoL2Supernode(t) }},
		{"NewTwoL2SupernodeInterop", func(t devtest.T) { NewTwoL2SupernodeInterop(t, 0) }},
		{"NewTwoL2SupernodeLightSequencerInterop", func(t devtest.T) { NewTwoL2SupernodeLightSequencerInterop(t, 0) }},
		{"NewTwoL2SupernodeFollowL2", func(t devtest.T) { NewTwoL2SupernodeFollowL2(t, 0) }},
		{"NewSingleSupernodeWithSyncTester", func(t devtest.T) { NewSingleSupernodeWithSyncTester(t) }},
	} {
		require.Truef(t, runSkipped(t, p.name, p.newPreset), "%s did not skip under kona-node", p.name)
	}
}

// These presets start the CL kind DEVSTACK_L2CL_KIND selects, so they must still run in the
// kona-node job.
func TestKonaCapablePresetsRunOnKonaNode(t *testing.T) {
	t.Setenv("DEVSTACK_L2CL_KIND", string(sysgo.MixedL2CLKona))
	for _, p := range []presetCase{
		{"NewMinimal", func(t devtest.T) { NewMinimal(t) }},
		{"NewMinimalNoFaultProofs", func(t devtest.T) { NewMinimalNoFaultProofs(t) }},
		{"NewMinimalWithConductors", func(t devtest.T) { NewMinimalWithConductors(t) }},
		{"NewMinimalWithKona", func(t devtest.T) { NewMinimalWithKona(t) }},
		{"NewSimpleWithSyncTester", func(t devtest.T) { NewSimpleWithSyncTester(t) }},
		{"NewSingleChainMultiNode", func(t devtest.T) { NewSingleChainMultiNode(t) }},
		{"NewSingleChainMultiNodeWithoutCheck", func(t devtest.T) { NewSingleChainMultiNodeWithoutCheck(t) }},
		{"NewSingleChainMultiNodeWithoutP2PWithoutCheck", func(t devtest.T) { NewSingleChainMultiNodeWithoutP2PWithoutCheck(t) }},
		{"NewSingleChainWithIsolatedVerifier", func(t devtest.T) { NewSingleChainWithIsolatedVerifier(t) }},
		{"NewSingleChainMultiNodeWithTestSeq", func(t devtest.T) { NewSingleChainMultiNodeWithTestSeq(t) }},
		{"NewSingleChainTwoVerifiersWithoutCheck", func(t devtest.T) { NewSingleChainTwoVerifiersWithoutCheck(t) }},
	} {
		require.Falsef(t, runSkipped(t, p.name, p.newPreset), "%s skipped under kona-node", p.name)
	}
}

func TestSkipOpNodeOnlyPresetOnKonaNodeSkipsOnlyKonaNode(t *testing.T) {
	for _, tc := range []struct {
		kind     sysgo.MixedL2CLKind
		wantSkip bool
	}{
		{"", false},
		{sysgo.MixedL2CLOpNode, false},
		{sysgo.MixedL2CLKona, true},
	} {
		t.Setenv("DEVSTACK_L2CL_KIND", string(tc.kind))
		require.Equalf(t, tc.wantSkip, runSkipped(t, "kind="+string(tc.kind), skipOpNodeOnlyPresetOnKonaNode),
			"DEVSTACK_L2CL_KIND=%q", tc.kind)
	}
}

// stopAtStartT stops a preset at its first temp dir, the JWT secret every runtime writes before
// it starts a node, so a test can see whether the preset skipped without starting a system.
type stopAtStartT struct{ devtest.T }

type presetStarted struct{}

func (stopAtStartT) TempDir() string                 { panic(presetStarted{}) }
func (stopAtStartT) TempDirWithPrefix(string) string { panic(presetStarted{}) }

// runSkipped runs fn in a subtest, stopped at its first temp dir, and reports whether the
// subtest skipped before reaching it.
func runSkipped(t *testing.T, name string, fn func(devtest.T)) bool {
	// With this set, devtest turns a skip into an error and carries on.
	t.Setenv(devtest.ExpectPreconditionsMet, "")
	var sub *testing.T
	t.Run(name, func(t *testing.T) {
		sub = t
		defer func() {
			if r := recover(); r != nil && r != (presetStarted{}) {
				panic(r)
			}
		}()
		fn(stopAtStartT{devtest.SerialT(t)})
	})
	return sub.Skipped()
}
