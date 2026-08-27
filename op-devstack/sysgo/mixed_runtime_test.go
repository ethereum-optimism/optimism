package sysgo

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
)

// TestOpRethDurationArg pins the two shapes op-reth's duration parser accepts. Go's own
// Duration.String renders "1.5s" and "2m0s", which the node rejects at startup with a clap error
// that only surfaces minutes later as an RPC timeout.
func TestOpRethDurationArg(t *testing.T) {
	for _, tc := range []struct {
		in   time.Duration
		want string
	}{
		{in: 25 * time.Millisecond, want: "25ms"},
		{in: 100 * time.Millisecond, want: "100ms"},
		{in: 1500 * time.Millisecond, want: "1500ms"},
		{in: time.Second, want: "1"},
		{in: 2 * time.Minute, want: "120"},
	} {
		t.Run(tc.want, func(t *testing.T) {
			require.Equal(t, tc.want, opRethDurationArg(tc.in))
		})
	}
}

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
