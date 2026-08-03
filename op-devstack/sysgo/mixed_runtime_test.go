package sysgo

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolveMixedL2ELOpts(t *testing.T) {
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
		{name: "args without a binary still apply", argsEnv: "--a=1", wantArgs: []string{"--a=1"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("DEVSTACK_L2EL_BINARY", tc.binaryEnv)
			t.Setenv("DEVSTACK_L2EL_ARGS", tc.argsEnv)

			opts := ResolveMixedL2ELOpts()

			if tc.wantBinary == "" && tc.wantArgs == nil {
				require.Empty(t, opts, "stock op-reth must be used when nothing is requested")
				return
			}

			cfg := DefaultOpRethConfig()
			for _, opt := range opts {
				opt.Apply(nil, ComponentTarget{}, cfg)
			}
			require.Equal(t, tc.wantBinary, cfg.Binary)
			require.Equal(t, tc.wantArgs, cfg.ExtraArgs)
			require.Equal(t, tc.wantBinary != "", cfg.DisableProofsHistory,
				"proofs history must be disabled for a superset binary and left alone otherwise")
		})
	}
}
