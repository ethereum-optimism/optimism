package presets

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/stretchr/testify/require"
)

type presetCase struct {
	name      string
	newPreset func(devtest.T)
}

// opNodeOnlyPresets start only op-node and op-supernode CLs, so under kona-node each must skip
// before it starts anything.
var opNodeOnlyPresets = []presetCase{
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
}

// konaCapablePresets start the CL kind DEVSTACK_L2CL_KIND selects, so they must still run in the
// kona-node job.
var konaCapablePresets = []presetCase{
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
}

// notPresets take a devtest.T but start nothing themselves.
var notPresets = []string{
	"NewMixedSingleChainFrontends", // wraps a runtime the caller already started
}

func TestOpNodeOnlyPresetsSkipOnKonaNode(t *testing.T) {
	t.Setenv("DEVSTACK_L2CL_KIND", string(sysgo.MixedL2CLKona))
	for _, p := range opNodeOnlyPresets {
		require.Truef(t, runSkipped(t, p.name, p.newPreset), "%s did not skip under kona-node", p.name)
	}
}

func TestKonaCapablePresetsRunOnKonaNode(t *testing.T) {
	t.Setenv("DEVSTACK_L2CL_KIND", string(sysgo.MixedL2CLKona))
	for _, p := range konaCapablePresets {
		require.Falsef(t, runSkipped(t, p.name, p.newPreset), "%s skipped under kona-node", p.name)
	}
}

// Every exported New* func whose first parameter is a devtest.T must be on exactly one of the three
// lists, so a new preset can't land without a kona-node decision. An unlisted op-node-only preset
// would repeat the op-node job under kona-node.
func TestEveryPresetIsClassified(t *testing.T) {
	var listed []string
	for _, p := range slices.Concat(opNodeOnlyPresets, konaCapablePresets) {
		listed = append(listed, p.name)
	}
	listed = append(listed, notPresets...)

	files, err := filepath.Glob("*.go")
	require.NoError(t, err)
	fset := token.NewFileSet()
	var constructors []string
	for _, path := range files {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, path, nil, 0)
		require.NoError(t, err)
		for _, decl := range f.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv != nil || !fn.Name.IsExported() || !strings.HasPrefix(fn.Name.Name, "New") {
				continue
			}
			if params := fn.Type.Params.List; len(params) > 0 && isDevtestT(params[0].Type) {
				constructors = append(constructors, fn.Name.Name)
			}
		}
	}
	require.NotEmpty(t, constructors)
	require.ElementsMatch(t, constructors, listed,
		"an extra in list A needs adding to one of opNodeOnlyPresets, konaCapablePresets and notPresets; "+
			"an extra in list B is listed twice or is not an exported New* func taking a devtest.T")
}

func isDevtestT(expr ast.Expr) bool {
	sel, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	pkg, ok := sel.X.(*ast.Ident)
	return ok && pkg.Name == "devtest" && sel.Sel.Name == "T"
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
