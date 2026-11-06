package sysgo

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func argsWithPrefix(args []string, prefix string) []string {
	var matches []string
	for _, arg := range args {
		if strings.HasPrefix(arg, prefix) {
			matches = append(matches, arg)
		}
	}
	return matches
}

func TestProofsHistoryNodeArgs(t *testing.T) {
	for _, tc := range []struct {
		name       string
		window     uint64
		wantWindow string
	}{
		{name: "zero window uses the runtime default", window: 0, wantWindow: "--proofs-history.window=10000"},
		{name: "explicit window replaces the default", window: 25, wantWindow: "--proofs-history.window=25"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			args := proofsHistoryNodeArgs(&OpRethConfig{ProofsHistoryWindow: tc.window}, "/proofs")

			require.Contains(t, args, "--proofs-history")
			require.Contains(t, args, "--proofs-history.storage-path=/proofs")
			require.Equal(t, []string{tc.wantWindow}, argsWithPrefix(args, "--proofs-history.window="),
				"op-reth rejects a repeated flag, so the window must appear exactly once")
		})
	}
}

func TestOpRethWithProofsHistoryWindow(t *testing.T) {
	cfg := DefaultOpRethConfig()
	OpRethWithProofsHistoryWindow(25).Apply(nil, ComponentTarget{}, cfg)

	require.Equal(t, uint64(25), cfg.ProofsHistoryWindow)
	require.Empty(t, cfg.ExtraArgs, "the window is a config field, not an extra arg")
}

func TestOpRethWithEthProofWindow(t *testing.T) {
	cfg := DefaultOpRethConfig()
	OpRethWithEthProofWindow(1_000_000).Apply(nil, ComponentTarget{}, cfg)

	require.Equal(t, []string{"--rpc.eth-proof-window=1000000"}, cfg.ExtraArgs)
}
