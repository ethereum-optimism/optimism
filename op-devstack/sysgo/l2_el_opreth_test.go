package sysgo

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/logpipe"
	"github.com/ethereum-optimism/optimism/op-service/ptr"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/stretchr/testify/require"
)

func TestOpRethWithEthProofWindow(t *testing.T) {
	cfg := DefaultOpRethConfig()
	OpRethWithEthProofWindow(1_000_000).Apply(nil, ComponentTarget{}, cfg)

	require.Equal(t, []string{"--rpc.eth-proof-window=1000000"}, cfg.ExtraArgs)
}

func TestOpRethWithPersistenceSchedule(t *testing.T) {
	for _, tc := range []struct {
		name         string
		schedule     OpRethPersistenceSchedule
		expectedArgs []string
	}{
		{
			name:         "reth_derived_backpressure",
			schedule:     OpRethPersistenceSchedule{MemoryBlockBufferTarget: 5, PersistenceThreshold: 50},
			expectedArgs: []string{"--engine.memory-block-buffer-target=5", "--engine.persistence-threshold=50"},
		},
		{
			name:     "pinned_backpressure",
			schedule: OpRethPersistenceSchedule{MemoryBlockBufferTarget: 5, PersistenceBackpressureThreshold: 8, PersistenceThreshold: 7},
			expectedArgs: []string{
				"--engine.memory-block-buffer-target=5",
				"--engine.persistence-backpressure-threshold=8",
				"--engine.persistence-threshold=7",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := DefaultOpRethConfig()
			OpRethWithPersistenceSchedule(tc.schedule).Apply(nil, ComponentTarget{}, cfg)

			require.Equal(t, tc.expectedArgs, cfg.ExtraArgs)
		})
	}
}

func TestOpRethPersistenceScheduleInterval(t *testing.T) {
	for _, tc := range []struct {
		persistenceThreshold uint64
		expectedInterval     uint64
	}{
		{persistenceThreshold: 5, expectedInterval: 1},
		{persistenceThreshold: 7, expectedInterval: 3},
		{persistenceThreshold: 50, expectedInterval: 46},
	} {
		t.Run(fmt.Sprintf("threshold_%d", tc.persistenceThreshold), func(t *testing.T) {
			schedule := OpRethPersistenceSchedule{MemoryBlockBufferTarget: 5, PersistenceThreshold: tc.persistenceThreshold}

			require.Equal(t, tc.expectedInterval, schedule.PersistenceInterval())
		})
	}
}

func TestTxPoolNodeArgs(t *testing.T) {
	for _, tc := range []struct {
		name     string
		cfg      OpRethConfig
		wantArgs []string
	}{
		{name: "defaults hold every origin to 1 wei", cfg: OpRethConfig{}, wantArgs: []string{"--txpool.minimum-priority-fee=1", "--txpool.nolocals"}},
		{name: "explicit minimum replaces the default", cfg: OpRethConfig{TxPoolMinimumPriorityFee: ptr.New(uint64(1_000_000_000))}, wantArgs: []string{"--txpool.minimum-priority-fee=1000000000", "--txpool.nolocals"}},
		{name: "explicit zero disables the minimum", cfg: OpRethConfig{TxPoolMinimumPriorityFee: ptr.New(uint64(0))}, wantArgs: []string{"--txpool.minimum-priority-fee=0", "--txpool.nolocals"}},
		{name: "local exemptions drop nolocals", cfg: OpRethConfig{TxPoolLocalExemptions: true}, wantArgs: []string{"--txpool.minimum-priority-fee=1"}},
		{name: "explicit minimum with local exemptions", cfg: OpRethConfig{TxPoolLocalExemptions: true, TxPoolMinimumPriorityFee: ptr.New(uint64(25))}, wantArgs: []string{"--txpool.minimum-priority-fee=25"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.wantArgs, txPoolNodeArgs(&tc.cfg), "unexpected transaction pool flags")
		})
	}
}

func TestOpRethWithTxPoolMinimumPriorityFee(t *testing.T) {
	for _, wei := range []uint64{0, 25} {
		t.Run(strconv.FormatUint(wei, 10), func(t *testing.T) {
			cfg := DefaultOpRethConfig()
			OpRethWithTxPoolMinimumPriorityFee(wei).Apply(nil, ComponentTarget{}, cfg)

			require.NotNil(t, cfg.TxPoolMinimumPriorityFee, "an explicit minimum, zero included, must be set")
			require.Equal(t, wei, *cfg.TxPoolMinimumPriorityFee)
			require.Empty(t, cfg.ExtraArgs, "the minimum is a config field, not an extra arg")
		})
	}
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

	require.True(t, cfg.TxConditional)
	require.Empty(t, cfg.ExtraArgs, "tx conditionals are a config field, not an extra arg")
}

func newStoppedOpReth(gt *testing.T, execPath string) *OpReth {
	return &OpReth{name: "el", execPath: execPath, p: devtest.SerialT(gt)}
}

// writeStubOpReth writes a binary that logs one error naming itself, announces its RPC endpoints
// the way op-reth does and then waits to be stopped. The error comes first so it has been observed
// by the time Start returns on the announcements.
func writeStubOpReth(gt *testing.T, name string) string {
	path := filepath.Join(gt.TempDir(), name)
	script := fmt.Sprintf(`#!/bin/sh
echo '{"level":"ERROR","fields":{"message":"hello from %s"}}'
echo '{"level":"INFO","fields":{"message":"RPC WS server started","url":"127.0.0.1:1"}}'
echo '{"level":"INFO","fields":{"message":"RPC auth server started","url":"127.0.0.1:2"}}'
exec /bin/sleep 1000
`, name)
	require.NoError(gt, os.WriteFile(path, []byte(script), 0o755))
	return path
}

func TestOpRethStartRunsTheSwitchedBinaryAndKeepsItsObservers(gt *testing.T) {
	// The stub never announces a metrics endpoint, which Start would wait for.
	gt.Setenv(sysgoMetricsEnabledEnvVar, "false")
	afterPath := writeStubOpReth(gt, "op-reth-after")
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
	node.SwitchBinary(afterPath)
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

	node.logObservers.notify(logpipe.StructuredRustLogEntry{Message: "first"})
	stop()
	node.logObservers.notify(logpipe.StructuredRustLogEntry{Message: "second"})

	require.Equal(gt, []string{"first"}, seen)
}

func TestAwaitAnnouncementReturnsTheAnnouncedValue(gt *testing.T) {
	sub, _ := newTestSubProcess(gt)
	require.NoError(gt, sub.Start("/bin/sh", []string{"-c", "exec /bin/sleep 30"}, nil))
	announced := make(chan string, 1)
	announced <- "ws://127.0.0.1:1"

	require.Equal(gt, "ws://127.0.0.1:1", awaitAnnouncement(sub.p, sub, newRecentLines(maxQuotedStderrLines), announced, "el", "user RPC"))
}

func TestAwaitAnnouncementReturnsWhatAnExitedProcessAnnounced(gt *testing.T) {
	sub, _ := newTestSubProcess(gt)
	require.NoError(gt, sub.Start("/bin/sh", []string{"-c", "exit 3"}, nil))
	announced := make(chan string, 1)
	announced <- "ws://127.0.0.1:1"
	<-sub.Exited()

	require.Equal(gt, "ws://127.0.0.1:1", awaitAnnouncement(sub.p, sub, newRecentLines(maxQuotedStderrLines), announced, "el", "user RPC"))
}

func TestAwaitAnnouncementFailsOnceTheProcessExitsWithoutAnnouncing(gt *testing.T) {
	sub, _ := newTestSubProcess(gt)
	require.NoError(gt, sub.Start("/bin/sh", []string{"-c", "exit 3"}, nil))
	announced := make(chan string, 1)

	require.PanicsWithValue(gt, "fail", func() {
		awaitAnnouncement(sub.p, sub, newRecentLines(maxQuotedStderrLines), announced, "el", "user RPC")
	})
}

func TestRequireRunningAfterAnnouncementsPassesWhileTheProcessRuns(gt *testing.T) {
	sub, _ := newTestSubProcess(gt)
	require.NoError(gt, sub.Start("/bin/sh", []string{"-c", "exec /bin/sleep 30"}, nil))

	require.NotPanics(gt, func() { requireRunningAfterAnnouncements(sub.p, sub, newRecentLines(maxQuotedStderrLines), "el") })
}

func TestRequireRunningAfterAnnouncementsFailsOnceTheProcessExited(gt *testing.T) {
	sub, _ := newTestSubProcess(gt)
	require.NoError(gt, sub.Start("/bin/sh", []string{"-c", "exit 3"}, nil))
	<-sub.Exited()

	require.PanicsWithValue(gt, "fail", func() { requireRunningAfterAnnouncements(sub.p, sub, newRecentLines(maxQuotedStderrLines), "el") })
}

func TestAwaitAnnouncementFailureQuotesTheUnstructuredStderr(gt *testing.T) {
	sub, capt := newTestSubProcess(gt)
	require.NoError(gt, sub.Start("/bin/sh", []string{"-c", "exit 1"}, nil))
	unstructuredStderr := newRecentLines(maxQuotedStderrLines)
	unstructuredStderr.add("Error: wrong msgpack marker FixArray(14)")

	require.PanicsWithValue(gt, "fail", func() {
		awaitAnnouncement(sub.p, sub, unstructuredStderr, make(chan string, 1), "el", "user RPC")
	})
	require.NotNil(gt, capt.FindLog(
		testlog.NewMessageContainsFilter("el exited during startup while waiting for its user RPC: exit status 1, last unstructured stderr lines:"),
		testlog.NewMessageContainsFilter("Error: wrong msgpack marker FixArray(14)"),
	))
}

func TestRecentLinesKeepsTheLastLines(gt *testing.T) {
	lines := newRecentLines(2)
	for _, line := range []string{"first", "second", "third"} {
		lines.add(line)
	}

	require.Equal(gt, []string{"second", "third"}, lines.snapshot())
}
