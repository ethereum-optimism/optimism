package sysgo

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
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
			args := proofsHistoryNodeArgs(tc.window, "/proofs")

			require.Contains(t, args, "--proofs-history")
			require.Contains(t, args, "--proofs-history.storage-path=/proofs")
			require.Equal(t, []string{tc.wantWindow}, argsWithPrefix(args, "--proofs-history.window="),
				"op-reth rejects a repeated flag, so the window must appear exactly once")
		})
	}
}

func TestProofsHistoryInitArgs(t *testing.T) {
	for _, tc := range []struct {
		name             string
		useSnapshot      bool
		window           uint64
		wantSnapshotFlag []string
		wantWindow       string
	}{
		{name: "snapshot backfill is op-reth's default and gets no flag", useSnapshot: true, window: 30, wantSnapshotFlag: nil, wantWindow: "--proofs-history.window=30"},
		{name: "plain backfill turns the snapshot off", useSnapshot: false, window: 30, wantSnapshotFlag: []string{"--proofs-history.use-snapshot=false"}, wantWindow: "--proofs-history.window=30"},
		{name: "zero window uses the runtime default", useSnapshot: true, window: 0, wantSnapshotFlag: nil, wantWindow: "--proofs-history.window=10000"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			args := proofsHistoryInitArgs("/genesis.json", "/data", "/proofs", tc.window, tc.useSnapshot)

			require.Equal(t, []string{"proofs", "init"}, args[:2])
			require.Contains(t, args, "--chain=/genesis.json")
			require.Contains(t, args, "--datadir=/data")
			require.Contains(t, args, "--proofs-history.storage-path=/proofs")
			require.Equal(t, []string{tc.wantWindow}, argsWithPrefix(args, "--proofs-history.window="))
			require.Equal(t, tc.wantSnapshotFlag, argsWithPrefix(args, "--proofs-history.use-snapshot"))
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

func TestOpRethNodeArgsCarryProofsHistoryOnlyOnceEnabled(t *testing.T) {
	baseArgs := []string{"node", "--datadir=/data"}
	n := &OpReth{args: baseArgs, proofsHistoryStoragePath: "/proofs", proofsHistoryWindow: 25}

	require.Equal(t, baseArgs, n.nodeArgs())

	n.proofsHistoryEnabled = true
	require.Equal(t, append([]string{"node", "--datadir=/data"}, proofsHistoryNodeArgs(25, "/proofs")...), n.nodeArgs())
	require.Equal(t, baseArgs, n.args, "the base arguments must stay free of the proofs-history flags")
}

func TestInitProofsHistoryEnablesProofsHistoryOnSuccessOnly(t *testing.T) {
	for _, tc := range []struct {
		name        string
		exitCode    int
		window      uint64
		wantEnabled bool
		wantWindow  uint64
	}{
		{name: "success enables proofs history with the init's window", exitCode: 0, window: 30, wantEnabled: true, wantWindow: 30},
		{name: "success keeps the configured window when the init names none", exitCode: 0, window: 0, wantEnabled: true, wantWindow: 25},
		{name: "failure leaves proofs history off and the window untouched", exitCode: 1, window: 30, wantEnabled: false, wantWindow: 25},
	} {
		t.Run(tc.name, func(t *testing.T) {
			n := newTestOpReth(t, fakeOpReth(t, tc.exitCode))

			report, err := n.InitProofsHistory(stack.ProofsHistoryInit{UseSnapshot: false, Window: tc.window})

			if tc.exitCode == 0 {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, "op-reth proofs init failed")
				require.ErrorContains(t, err, "proofs init --chain=/genesis.json", "the error must carry the subcommand's output")
			}
			require.Contains(t, report.Output, "--proofs-history.use-snapshot=false", "the fake binary echoes the arguments it got")
			require.Positive(t, report.Duration)
			require.Equal(t, tc.wantEnabled, n.proofsHistoryEnabled)
			require.Equal(t, tc.wantWindow, n.proofsHistoryWindow)
		})
	}
}

// fakeOpReth writes a script that echoes its arguments and exits with exitCode, standing in for
// the op-reth binary.
func fakeOpReth(t *testing.T, exitCode int) string {
	path := filepath.Join(t.TempDir(), "op-reth")
	script := fmt.Sprintf("#!/bin/sh\necho \"$@\"\nexit %d\n", exitCode)
	require.NoError(t, os.WriteFile(path, []byte(script), 0o755))
	return path
}

func newTestOpReth(t *testing.T, execPath string) *OpReth {
	return &OpReth{
		chainConfigPath:          "/genesis.json",
		dataDirPath:              "/data",
		execPath:                 execPath,
		name:                     "el",
		p:                        devtest.SerialT(t),
		proofsHistoryStoragePath: "/proofs",
		proofsHistoryWindow:      25,
	}
}
