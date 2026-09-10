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

func TestOpRethWithOtterscanAPI(t *testing.T) {
	cfg := DefaultOpRethConfig()
	require.False(t, cfg.EnableOtterscanAPI)
	require.NotContains(t, opRethRPCModules(cfg), "ots")

	OpRethWithOtterscanAPI().Apply(nil, ComponentTarget{}, cfg)

	require.True(t, cfg.EnableOtterscanAPI)
	require.Equal(t, "admin,debug,eth,net,trace,txpool,web3,rpc,reth,miner,ots", opRethRPCModules(cfg))
}
