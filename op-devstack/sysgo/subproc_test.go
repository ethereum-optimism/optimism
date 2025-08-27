package sysgo

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-service/logpipe"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

func TestSubProcess(gt *testing.T) {
	tLog := testlog.Logger(gt, log.LevelInfo)
	logger, capt := testlog.CaptureLogger(gt, log.LevelInfo)

	onFailNow := func(v bool) {
		panic("fail")
	}
	onSkipNow := func() {
		panic("skip")
	}
	p := devtest.NewP(context.Background(), logger, onFailNow, onSkipNow)
	gt.Cleanup(p.Close)

	logProc := logpipe.LogProcessor(func(line []byte) {
		logger.Info(string(line))
		tLog.Info("Sub-process logged message", "line", string(line))
	})
	sp := NewSubProcess(p, logProc, logProc)

	gt.Log("Running first sub-process")
	testSleep(gt, capt, sp)
	gt.Log("Restarting, second run")
	capt.Clear()
	testSleep(gt, capt, sp)
	gt.Log("Trying a different command now")
	capt.Clear()
	testEcho(gt, capt, sp)
	gt.Log("Second run of different command")
	capt.Clear()
	testEcho(gt, capt, sp)
}

// testEcho tests that we can handle a sub-process that completes on its own
func testEcho(gt *testing.T, capt *testlog.CapturingHandler, sp *SubProcess) {
	require.NoError(gt, sp.Start("/bin/echo", []string{"hello world"}, []string{}))
	gt.Log("Started sub-process")
	require.NoError(gt, sp.Wait(context.Background()), "echo must complete")
	require.NoError(gt, sp.Stop())
	gt.Log("Stopped sub-process")

	require.NotNil(gt, capt.FindLog(
		testlog.NewMessageFilter("hello world")))

	require.NotNil(gt, capt.FindLog(
		testlog.NewMessageFilter("Sub-process gracefully exited")))
}

// testSleep tests that we can force shut down a sub-process that is stuck
func testSleep(gt *testing.T, capt *testlog.CapturingHandler, sp *SubProcess) {
	// Sleep for very, very, long
	require.NoError(gt, sp.Start("/bin/sleep", []string{"10000000000"}, []string{}))
	gt.Log("Started sub-process")
	// Shut down the process before the sleep completes
	require.NoError(gt, sp.Kill())
	gt.Log("Killed sub-process")

	require.NotNil(gt, capt.FindLog(
		testlog.NewMessageFilter("Sub-process did not respond to interrupt, force-closing now")))

	require.NotNil(gt, capt.FindLog(
		testlog.NewMessageFilter("Successfully force-closed sub-process")))
}
