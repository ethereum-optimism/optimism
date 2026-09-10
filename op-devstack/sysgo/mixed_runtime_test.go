package sysgo

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
)

func TestMixedL2ELOptsFromEnv(t *testing.T) {
	for _, tc := range []struct {
		name       string
		binaryEnv  string
		argsEnv    string
		wantBinary string
		wantArgs   []string
	}{
		{name: "unset yields no options"},
		{name: "empty yields no options", binaryEnv: "", argsEnv: ""},
		{name: "binary name selects that binary", binaryEnv: "op-reth-superset", wantBinary: "op-reth-superset"},
		{name: "args reach the CLI", binaryEnv: "op-reth-superset", argsEnv: "--extra.bind=127.0.0.1:0", wantBinary: "op-reth-superset", wantArgs: []string{"--extra.bind=127.0.0.1:0"}},
		{name: "multiple args split on whitespace", binaryEnv: "op-reth-superset", argsEnv: "  --a=1   --b=2 ", wantBinary: "op-reth-superset", wantArgs: []string{"--a=1", "--b=2"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			opts, err := mixedL2ELOptsFromEnv(tc.binaryEnv, tc.argsEnv)
			require.NoError(t, err)

			if tc.wantBinary == "" {
				require.Empty(t, opts, "stock op-reth must be used when no override is requested")
				return
			}

			cfg := DefaultOpRethConfig()
			for _, opt := range opts {
				opt.Apply(nil, ComponentTarget{}, cfg)
			}
			require.Equal(t, tc.wantBinary, cfg.Binary)
			require.Equal(t, tc.wantArgs, cfg.ExtraArgs)
			require.True(t, cfg.DisableProofsHistory,
				"proofs history must be disabled for a superset binary")
		})
	}
}

// Args only qualify an override binary, so naming them alone is a misconfiguration: silently
// dropping them would let a stale value in the environment look like it took effect.
func TestMixedL2ELOptsFromEnvRejectsArgsWithoutBinary(t *testing.T) {
	opts, err := mixedL2ELOptsFromEnv("", "--a=1")

	require.ErrorContains(t, err, devstackL2ELOverrideArgsEnv)
	require.Empty(t, opts)
}

// The exported resolver must read the documented variable names.
func TestResolveMixedL2ELOptsReadsEnv(t *testing.T) {
	t.Setenv(devstackL2ELOverrideBinaryEnv, "op-reth-superset")
	t.Setenv(devstackL2ELOverrideArgsEnv, "--extra.bind=127.0.0.1:0")

	cfg := DefaultOpRethConfig()
	for _, opt := range ResolveMixedL2ELOpts(devtest.SerialT(t)) {
		opt.Apply(nil, ComponentTarget{}, cfg)
	}

	require.Equal(t, "op-reth-superset", cfg.Binary)
	require.Equal(t, []string{"--extra.bind=127.0.0.1:0"}, cfg.ExtraArgs)
	require.True(t, cfg.DisableProofsHistory)
}

func TestMixedCLPeeringTracksFactoryHandlingPerSlot(t *testing.T) {
	stock := mixedSingleChainNode{}
	external := mixedSingleChainNode{factoryHandledCL: true}

	require.True(t, shouldConnectMixedCLPeers(stock, stock),
		"no factory, or a factory that declines every slot, preserves stock peering")
	require.False(t, shouldConnectMixedCLPeers(stock, external),
		"a mixed stock/external edge must not assume the external CL implements op-node admin RPCs")
	require.False(t, shouldConnectMixedCLPeers(external, stock))
	require.False(t, shouldConnectMixedCLPeers(external, external))
}

func TestMixedUnsafeSourceIsIndependentOfSpecOrder(t *testing.T) {
	verifier := MixedSingleChainNodeSpec{CLKey: "verifier"}
	sequencer := MixedSingleChainNodeSpec{CLKey: "sequencer", IsSequencer: true}
	nodes := []mixedSingleChainNode{
		{spec: verifier},
		{spec: sequencer},
	}

	source := mixedUnsafeSourceNode(nodes, verifier)
	require.NotNil(t, source)
	require.Equal(t, "sequencer", source.spec.CLKey)
	require.Nil(t, mixedUnsafeSourceNode(nodes, sequencer))

	isolatedVerifier := verifier
	isolatedVerifier.IsolateFromL2P2P = true
	require.Nil(t, mixedUnsafeSourceNode(nodes, isolatedVerifier))

	isolatedSequencer := sequencer
	isolatedSequencer.IsolateFromL2P2P = true
	nodes[1].spec = isolatedSequencer
	require.Nil(t, mixedUnsafeSourceNode(nodes, verifier),
		"an isolated sequencer cannot become another node's unsafe source")
}

func TestMixedFactoryStartsSequencersBeforeVerifiers(t *testing.T) {
	nodes := []mixedSingleChainNode{
		{spec: MixedSingleChainNodeSpec{CLKey: "verifier-a"}},
		{spec: MixedSingleChainNodeSpec{CLKey: "sequencer", IsSequencer: true}},
		{spec: MixedSingleChainNodeSpec{CLKey: "verifier-b"}},
	}

	require.Equal(t, []int{1, 0, 2}, mixedCLStartOrder(nodes))
}

func TestMixedDeclinedVerifierFollowsOnlyExternalSourceCL(t *testing.T) {
	source := &mixedSingleChainNode{
		el: &fakeFactoryL2EL{rpc: "http://127.0.0.1:8545"},
		cl: &fakeExternalL2CL{rpc: "http://127.0.0.1:9545"},
	}

	require.Empty(t, mixedNodeFollowSource(source),
		"a factory that declines all slots retains stock P2P-only follow behavior")
	source.factoryHandledCL = true
	require.Equal(t, source.cl.UserRPC(), mixedNodeFollowSource(source),
		"a declined stock verifier follows an external source through its rollup RPC")
}
