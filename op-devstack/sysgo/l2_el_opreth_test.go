package sysgo

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/logpipe"
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

func TestTxPoolNodeArgs(t *testing.T) {
	for _, tc := range []struct {
		name     string
		cfg      OpRethConfig
		wantArgs []string
	}{
		{name: "defaults hold every origin to 1 wei", cfg: OpRethConfig{}, wantArgs: []string{"--txpool.minimum-priority-fee=1", "--txpool.nolocals"}},
		{name: "explicit minimum replaces the default", cfg: OpRethConfig{TxPoolMinimumPriorityFee: 1_000_000_000}, wantArgs: []string{"--txpool.minimum-priority-fee=1000000000", "--txpool.nolocals"}},
		{name: "local exemptions drop nolocals", cfg: OpRethConfig{TxPoolLocalExemptions: true}, wantArgs: []string{"--txpool.minimum-priority-fee=1"}},
		{name: "explicit minimum with local exemptions", cfg: OpRethConfig{TxPoolLocalExemptions: true, TxPoolMinimumPriorityFee: 25}, wantArgs: []string{"--txpool.minimum-priority-fee=25"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.wantArgs, txPoolNodeArgs(&tc.cfg), "unexpected transaction pool flags")
		})
	}
}

func TestOpRethWithTxPoolMinimumPriorityFee(t *testing.T) {
	cfg := DefaultOpRethConfig()
	OpRethWithTxPoolMinimumPriorityFee(25).Apply(nil, ComponentTarget{}, cfg)

	require.Equal(t, uint64(25), cfg.TxPoolMinimumPriorityFee)
	require.Empty(t, cfg.ExtraArgs, "the minimum is a config field, not an extra arg")
}

func TestOpRethWithTxPoolLocalExemptions(t *testing.T) {
	cfg := DefaultOpRethConfig()
	OpRethWithTxPoolLocalExemptions().Apply(nil, ComponentTarget{}, cfg)

	require.True(t, cfg.TxPoolLocalExemptions)
	require.Empty(t, cfg.ExtraArgs, "the exemptions are a config field, not an extra arg")
}

func TestOpRethWithTxConditional(t *testing.T) {
	cfg := DefaultOpRethConfig()
	OpRethWithTxConditional().Apply(nil, ComponentTarget{}, cfg)

	require.Equal(t, []string{"--rollup.enable-tx-conditional"}, cfg.ExtraArgs)
}

func newStoppedOpReth(gt *testing.T, execPath string) *OpReth {
	return &OpReth{name: "el", execPath: execPath, p: devtest.SerialT(gt)}
}

// writeStubOpReth writes a binary that announces its RPC endpoints the way op-reth does, logs one
// error naming itself and then waits to be stopped.
func writeStubOpReth(gt *testing.T, name string) string {
	path := filepath.Join(gt.TempDir(), name)
	script := fmt.Sprintf(`#!/bin/sh
echo '{"level":"INFO","fields":{"message":"RPC WS server started","url":"127.0.0.1:1"}}'
echo '{"level":"INFO","fields":{"message":"RPC auth server started","url":"127.0.0.1:2"}}'
echo '{"level":"ERROR","fields":{"message":"hello from %s"}}'
exec /bin/sleep 1000
`, name)
	require.NoError(gt, os.WriteFile(path, []byte(script), 0o755))
	return path
}

func TestOpRethStartRunsTheSwitchedBinaryAndKeepsItsObservers(gt *testing.T) {
	gt.Setenv("RUST_BINARY_PATH_OP_RETH_AFTER", writeStubOpReth(gt, "op-reth-after"))
	node := newStoppedOpReth(gt, writeStubOpReth(gt, "op-reth-before"))
	var mu sync.Mutex
	var errorsSeen []string
	stop := node.ObserveLogs(func(e logpipe.LogEntry) {
		if e.LogLevel() < log.LevelError {
			return
		}
		mu.Lock()
		defer mu.Unlock()
		errorsSeen = append(errorsSeen, e.LogMessage())
	})
	defer stop()

	node.Start()
	node.Stop()
	node.SwitchBinary("op-reth-after")
	node.Start()
	node.Stop()

	mu.Lock()
	defer mu.Unlock()
	require.Equal(gt, []string{"hello from op-reth-before", "hello from op-reth-after"}, errorsSeen)
}

func TestOpRethObserveLogsUntilStopped(gt *testing.T) {
	node := newStoppedOpReth(gt, "/bin/op-reth")
	var seen []string
	stop := node.ObserveLogs(func(e logpipe.LogEntry) { seen = append(seen, e.LogMessage()) })

	node.notifyLogObservers(logpipe.StructuredRustLogEntry{Message: "first"})
	stop()
	node.notifyLogObservers(logpipe.StructuredRustLogEntry{Message: "second"})

	require.Equal(gt, []string{"first"}, seen)
}

func TestRaiseUnstructuredStderr(gt *testing.T) {
	panicLine := raiseUnstructuredStderr(logpipe.ParseRustStructuredLogs([]byte("thread 'main' panicked at src/main.rs")))
	require.Equal(gt, log.LevelError, panicLine.LogLevel())

	structured := raiseUnstructuredStderr(logpipe.ParseRustStructuredLogs([]byte(`{"level":"WARN","fields":{"message":"x"}}`)))
	require.Equal(gt, log.LevelWarn, structured.LogLevel())
}

func TestAwaitAnnouncementReturnsTheAnnouncedValue(gt *testing.T) {
	sub, _ := newTestSubProcess(gt)
	require.NoError(gt, sub.Start("/bin/sh", []string{"-c", "exec /bin/sleep 30"}, nil))
	announced := make(chan string, 1)
	announced <- "ws://127.0.0.1:1"

	require.Equal(gt, "ws://127.0.0.1:1", awaitAnnouncement(sub.p, sub, announced, "el", "user RPC"))
}

func TestAwaitAnnouncementFailsOnceTheProcessExitsWithoutAnnouncing(gt *testing.T) {
	sub, _ := newTestSubProcess(gt)
	require.NoError(gt, sub.Start("/bin/sh", []string{"-c", "exit 3"}, nil))
	announced := make(chan string, 1)

	require.PanicsWithValue(gt, "fail", func() { awaitAnnouncement(sub.p, sub, announced, "el", "user RPC") })
}
