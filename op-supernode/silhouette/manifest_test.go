package silhouette

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	opnodecfg "github.com/ethereum-optimism/optimism/op-node/config"
	rollupNode "github.com/ethereum-optimism/optimism/op-node/node"
	"github.com/ethereum-optimism/optimism/op-node/p2p"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
)

// writeManifest lays out a manifest and the verifier config it names, the way an operator would:
// two files in one directory, the second referenced by a relative path.
func writeManifest(t *testing.T, entry ManifestChain, verifier *Config) string {
	t.Helper()
	dir := t.TempDir()
	vpath := filepath.Join(dir, "verifier.json")
	raw, err := json.Marshal(verifier)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(vpath, raw, 0o600))

	entry.VerifierConfig = "verifier.json"
	mraw, err := json.Marshal(Manifest{Chains: []ManifestChain{entry}})
	require.NoError(t, err)
	mpath := filepath.Join(dir, "silhouette.json")
	require.NoError(t, os.WriteFile(mpath, mraw, 0o600))
	return mpath
}

func TestManifestLoadsAndResolvesRelativeConfigPaths(t *testing.T) {
	env := newTestEnv(t, l1GenesisNum+10)
	path := writeManifest(t, ManifestChain{ChainID: 424250, Labels: "derivation"}, env.cfg)

	m, err := LoadManifest(path)
	require.NoError(t, err)

	decl, ok := m.Lookup(eth.ChainIDFromUInt64(424250))
	require.True(t, ok)
	require.NotNil(t, decl.Config(), "the verifier config is loaded at manifest-load time, not lazily")
	require.Equal(t, env.cfg.Submitter, decl.Config().Submitter)
	require.Equal(t, env.cfg.Anchor.OutputRoot, decl.Config().Anchor.OutputRoot)

	labels, err := decl.LabelSource()
	require.NoError(t, err)
	require.Equal(t, LabelsFromDerivation, labels)

	// A chain the manifest says nothing about is an ordinary driven chain, and the lookup says so.
	_, ok = m.Lookup(eth.ChainIDFromUInt64(424246))
	require.False(t, ok)
}

// TestManifestDefaultsToTheVerifierPosture pins the direction of the default. An omitted posture
// must not be able to turn a public verifier into a node that labels P from facts it did not
// derive, so "" means derivation.
func TestManifestDefaultsToTheVerifierPosture(t *testing.T) {
	env := newTestEnv(t, l1GenesisNum+10)
	path := writeManifest(t, ManifestChain{ChainID: 424250}, env.cfg)
	m, err := LoadManifest(path)
	require.NoError(t, err)
	decl, ok := m.Lookup(eth.ChainIDFromUInt64(424250))
	require.True(t, ok)
	labels, err := decl.LabelSource()
	require.NoError(t, err)
	require.Equal(t, LabelsFromDerivation, labels)
}

func TestManifestSequencerPosture(t *testing.T) {
	env := newTestEnv(t, l1GenesisNum+10)
	path := writeManifest(t, ManifestChain{ChainID: 424250, Labels: "proven-head"}, env.cfg)
	m, err := LoadManifest(path)
	require.NoError(t, err)
	decl, _ := m.Lookup(eth.ChainIDFromUInt64(424250))
	labels, err := decl.LabelSource()
	require.NoError(t, err)
	require.Equal(t, LabelsFromProvenHead, labels)
}

func TestManifestRejections(t *testing.T) {
	env := newTestEnv(t, l1GenesisNum+10)
	dir := t.TempDir()
	vpath := filepath.Join(dir, "verifier.json")
	raw, err := json.Marshal(env.cfg)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(vpath, raw, 0o600))

	write := func(t *testing.T, body string) string {
		t.Helper()
		p := filepath.Join(t.TempDir(), "silhouette.json")
		require.NoError(t, os.WriteFile(p, []byte(body), 0o600))
		return p
	}

	t.Run("unknown field", func(t *testing.T) {
		p := write(t, `{"chains":[{"chainID":1,"verifierConfig":"v.json","mode":"fast"}]}`)
		_, err := LoadManifest(p)
		require.ErrorContains(t, err, "mode")
	})
	t.Run("no chains", func(t *testing.T) {
		p := write(t, `{"chains":[]}`)
		_, err := LoadManifest(p)
		require.ErrorContains(t, err, "declares no chains")
	})
	t.Run("duplicate chain", func(t *testing.T) {
		p := write(t, `{"chains":[{"chainID":7,"verifierConfig":"`+vpath+`"},{"chainID":7,"verifierConfig":"`+vpath+`"}]}`)
		_, err := LoadManifest(p)
		require.ErrorContains(t, err, "declared twice")
	})
	t.Run("unknown posture", func(t *testing.T) {
		p := write(t, `{"chains":[{"chainID":7,"labels":"receipts","verifierConfig":"`+vpath+`"}]}`)
		_, err := LoadManifest(p)
		require.ErrorContains(t, err, "unknown labels posture")
	})
	t.Run("missing verifier config", func(t *testing.T) {
		p := write(t, `{"chains":[{"chainID":7}]}`)
		_, err := LoadManifest(p)
		require.ErrorContains(t, err, "no verifierConfig")
	})
	t.Run("unreadable verifier config", func(t *testing.T) {
		p := write(t, `{"chains":[{"chainID":7,"verifierConfig":"/nonexistent/v.json"}]}`)
		_, err := LoadManifest(p)
		require.ErrorContains(t, err, "read silhouette config")
	})
}

// TestManifestRejectsChainsTheSupernodeIsNotRunning is the check that turns a typo into a refusal
// to start. A manifest entry for an unconfigured chain is silently inert otherwise, and the likely
// intent was to run it.
func TestManifestRejectsChainsTheSupernodeIsNotRunning(t *testing.T) {
	env := newTestEnv(t, l1GenesisNum+10)
	path := writeManifest(t, ManifestChain{ChainID: 424250}, env.cfg)
	m, err := LoadManifest(path)
	require.NoError(t, err)

	require.NoError(t, m.CheckChains([]uint64{424246, 424250}))
	require.ErrorContains(t, m.CheckChains([]uint64{424246}), "not in --chains")
	// A nil manifest is the no-silhouette-chains case and must never be an error.
	require.NoError(t, (*Manifest)(nil).CheckChains([]uint64{1}))
}

func TestL1ChainConfigLookup(t *testing.T) {
	sep, err := L1ChainConfig(&Config{L1ChainID: 11155111})
	require.NoError(t, err)
	require.Equal(t, uint64(11155111), sep.ChainID.Uint64())

	mainnet, err := L1ChainConfig(&Config{L1ChainID: 1})
	require.NoError(t, err)
	require.Equal(t, uint64(1), mainnet.ChainID.Uint64())

	// An unknown L1 is an ERROR, not a default. The value decides whether the L1 block's excess
	// blob gas is priced under Cancun or Prague, and that number goes into the L1-info transaction
	// — so a guess would be a fabricated consensus-relevant value, which is the one thing the
	// fabrication classes forbid outright.
	_, err = L1ChainConfig(&Config{L1ChainID: 900})
	require.ErrorContains(t, err, "no known L1 chain config")
	require.ErrorContains(t, err, "l1ChainConfigPath", "the error must say how to answer it")
}

// TestL1ChainConfigFromFile covers the local-cluster case: a devnet L1 has no public constant to
// look up, so the operator names it. Found by an agent standing the switch up on a devstack cluster,
// where L1 is chain 900 and the supernode refused to start.
func TestL1ChainConfigFromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "l1-chain.json")
	raw, err := json.Marshal(params.SepoliaChainConfig)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, raw, 0o600))

	cfg, err := L1ChainConfig(&Config{L1ChainID: 11155111, L1ChainConfigPath: path})
	require.NoError(t, err)
	require.Equal(t, uint64(11155111), cfg.ChainID.Uint64())

	// A file for the wrong chain is refused rather than used: it is the likeliest way to get this
	// wrong, and the failure it would otherwise cause is a bad blob-fee field in a consensus
	// transaction, which surfaces nowhere near the config.
	_, err = L1ChainConfig(&Config{L1ChainID: 900, L1ChainConfigPath: path})
	require.ErrorContains(t, err, "but this chain settles on 900")

	_, err = L1ChainConfig(&Config{L1ChainID: 900, L1ChainConfigPath: filepath.Join(dir, "nope.json")})
	require.ErrorContains(t, err, "read L1 chain config")
}

// TestAssembleTakesNoGossip pins the safety property that a silhouette chain is not reachable over
// P2P. A payload gossiped at a height the shim HOLDS, with the same parent and a different hash, is
// the shape the shim treats as a fatal disagreement about proven history and halts on permanently
// (G3 D4) -- correct when the claim comes from derivation, absurd when it comes from a stranger on
// a gossip topic. The door is closed rather than guarded.
func TestAssembleTakesNoGossip(t *testing.T) {
	env := newTestEnv(t, l1GenesisNum+10)
	vncfg := &opnodecfg.Config{Rollup: *silhouetteRollupConfig()}
	vncfg.P2P = &p2p.Config{}
	vncfg.P2PSigner = &p2p.PreparedSigner{}

	a, err := Assemble(testlog.Logger(t, log.LevelError), env.cfg, AssemblyConfig{
		Rollup: vncfg, L1Chain: sepoliaChainConfig(), SysCfg: vncfg.Rollup.Genesis.SystemConfig,
		L1: env.l1, Blobs: env.blobs, L1Headers: env.l1, Labels: LabelsFromDerivation,
	}, vncfg, &rollupNode.InitializationOverrides{})
	require.NoError(t, err)
	defer a.Close()

	require.Nil(t, vncfg.P2P, "a proof-carried chain has nothing to learn from a peer")
	require.Nil(t, vncfg.P2PSigner)
	require.False(t, vncfg.P2PEnabled())
}

// TestAssembleWiresBothSeams is the phase-1 claim in one test: after Assemble, the caller's
// virtual-node config and initialization overrides are the ONLY things that changed, and the stock
// container built from them will talk to the shim and read the proof-batch source.
func TestAssembleWiresBothSeams(t *testing.T) {
	env := newTestEnv(t, l1GenesisNum+10)
	logger := testlog.Logger(t, log.LevelError)

	vncfg := &opnodecfg.Config{Rollup: *silhouetteRollupConfig()}
	// A sequencer-enabled config, so the pinning below is observed rather than assumed.
	vncfg.Driver.SequencerEnabled = true
	vncfg.Driver.SequencerStopped = false
	overrides := &rollupNode.InitializationOverrides{}

	a, err := Assemble(logger, env.cfg, AssemblyConfig{
		Rollup:    vncfg,
		L1Chain:   sepoliaChainConfig(),
		SysCfg:    vncfg.Rollup.Genesis.SystemConfig,
		L1:        env.l1,
		Blobs:     env.blobs,
		L1Headers: env.l1,
		Labels:    LabelsFromDerivation,
	}, vncfg, overrides)
	require.NoError(t, err)
	defer a.Close()

	require.Equal(t, eth.ChainIDFromUInt64(424250), a.ChainID)

	// Seam 1: the execution client is the shim, reachable in-process.
	prepared, ok := vncfg.L2.(*opnodecfg.PreparedL2Endpoints)
	require.True(t, ok, "L2 endpoint must be the in-process shim client")
	require.NotNil(t, prepared.Client)
	var chainIDHex string
	require.NoError(t, prepared.Client.CallContext(t.Context(), &chainIDHex, "eth_chainId"))
	require.Equal(t, "0x6793a", chainIDHex, "the shim answers for P over the wired-in client")

	// Seam 2: the derivation input is the proof-batch source, and it is the SAME source object the
	// assembly holds — so the log sink attached later lands on the source the pipeline reads.
	require.Same(t, a.Source, overrides.DataSourceOverride)

	// A silhouette chain never sequences, whatever the config said.
	require.False(t, vncfg.Driver.SequencerEnabled)
	require.True(t, vncfg.Driver.SequencerStopped)

	// The shim and the source share one fact table. Handing them different ones is the bug this
	// construction order exists to make impossible, so it is asserted rather than described.
	require.Same(t, a.Facts, a.Source.facts)
	require.Same(t, a.Facts, a.Shim.facts)

	// Seam 3: the query surface embedding the shim would otherwise have removed from the network.
	ns := map[string]bool{}
	for _, api := range overrides.ExtraAPIs {
		ns[api.Namespace] = true
	}
	require.True(t, ns["eth"], "P's blocks must be queryable at the chain's own route")
	require.True(t, ns["silhouette"], "G2 D8: self-declaration lives at the service layer, so it must be reachable")
	require.False(t, ns["engine"],
		"the Engine API is the private channel to this chain's execution client and must not be published")
}

// TestAssembleDoesNotPublishTheEngineAPI states the exclusion on its own, because it is a security
// property rather than a tidiness one: a published newPayload for a chain nobody may extend is an
// invitation with no legitimate caller, and the shim's fail-stop is a guard, not a reason to offer
// the door.
func TestAssembleDoesNotPublishTheEngineAPI(t *testing.T) {
	env := newTestEnv(t, l1GenesisNum+10)
	shim := NewShim(testlog.Logger(t, log.LevelError), silhouetteRollupConfig(), sepoliaChainConfig(),
		silhouetteRollupConfig().Genesis.SystemConfig, env.l1, env.facts)

	all := map[string]bool{}
	for _, api := range shim.APIs() {
		all[api.Namespace] = true
	}
	public := map[string]bool{}
	for _, api := range shim.PublicAPIs() {
		public[api.Namespace] = true
	}
	require.True(t, all["engine"], "the shim still speaks the Engine API to its own node")
	require.False(t, public["engine"])
	// Everything else the shim serves is public: the public set is the full set minus engine, so a
	// namespace added later is published by default rather than forgotten.
	require.Len(t, public, len(all)-1)
}

func TestAssembleRejectsMissingInputs(t *testing.T) {
	env := newTestEnv(t, l1GenesisNum+10)
	logger := testlog.Logger(t, log.LevelError)
	vncfg := &opnodecfg.Config{Rollup: *silhouetteRollupConfig()}

	_, err := Assemble(logger, env.cfg, AssemblyConfig{Rollup: vncfg}, nil, nil)
	require.ErrorContains(t, err, "needs a virtual-node config")

	_, err = Assemble(logger, env.cfg, AssemblyConfig{}, vncfg, &rollupNode.InitializationOverrides{})
	require.ErrorContains(t, err, "needs P's rollup config")

	bad := *env.cfg
	bad.Inbox = bad.Submitter
	_, err = Assemble(logger, &bad, AssemblyConfig{Rollup: vncfg}, vncfg, &rollupNode.InitializationOverrides{})
	require.ErrorContains(t, err, "silhouette config")
}
