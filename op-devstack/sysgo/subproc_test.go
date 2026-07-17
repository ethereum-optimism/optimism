package sysgo

import (
	"context"
	"testing"
	"time"

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

	logCallback := logpipe.LogCallback(func(line []byte) {
		logger.Info(string(line))
		tLog.Info("Sub-process logged message", "line", string(line))
	})
	sp := NewSubProcess(p, logCallback, logCallback)

	gt.Log("Running first sub-process")
	testSleep(gt, capt, sp)
	gt.Log("Restarting, second run")
	capt.Clear()
	testSleep(gt, capt, sp)
	gt.Log("Force-killing and restarting")
	capt.Clear()
	testKill(gt, capt, sp)
	gt.Log("Trying a different command now")
	capt.Clear()
	testEcho(gt, capt, sp)
	gt.Log("Second run of different command")
	capt.Clear()
	testEcho(gt, capt, sp)
}

func TestSubProcessConcurrentStop(gt *testing.T) {
	tLog := testlog.Logger(gt, log.LevelInfo)
	logger, _ := testlog.CaptureLogger(gt, log.LevelInfo)

	onFailNow := func(v bool) {
		panic("fail")
	}
	onSkipNow := func() {
		panic("skip")
	}
	p := devtest.NewP(context.Background(), logger, onFailNow, onSkipNow)
	gt.Cleanup(p.Close)

	logCallback := logpipe.LogCallback(func(line []byte) {
		logger.Info(string(line))
		tLog.Info("Sub-process logged message", "line", string(line))
	})
	sp := NewSubProcess(p, logCallback, logCallback)

	require.NoError(gt, sp.Start("/bin/sh", []string{"-c", "trap '' INT; exec /bin/sleep 10000000000"}, []string{}))

	start := make(chan struct{})
	errCh := make(chan error, 2)
	for i := 0; i < 2; i++ {
		go func() {
			<-start
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			errCh <- sp.StopControlled(ctx, 100*time.Millisecond, time.Second)
		}()
	}
	close(start)

	require.NoError(gt, <-errCh)
	require.NoError(gt, <-errCh)

	require.NoError(gt, sp.Start("/bin/echo", []string{"restart ok"}, []string{}))
	require.NoError(gt, sp.Stop(false))
}

func testKill(gt *testing.T, capt *testlog.CapturingHandler, sp *SubProcess) {
	require.NoError(gt, sp.Start("/bin/sleep", []string{"10000000000"}, []string{}))
	require.NoError(gt, sp.Kill())
	require.NotNil(gt, capt.FindLog(testlog.NewMessageFilter("Killing sub-process")))

	require.NoError(gt, sp.Start("/bin/echo", []string{"restart after kill"}, []string{}))
	require.NoError(gt, sp.Stop(false))
}

// testEcho tests that we can handle a sub-process that completes on its own
func testEcho(gt *testing.T, capt *testlog.CapturingHandler, sp *SubProcess) {
	require.NoError(gt, sp.Start("/bin/echo", []string{"hello world"}, []string{}))
	gt.Log("Started sub-process")
	require.NoError(gt, sp.Stop(false))
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
	require.NoError(gt, sp.Stop(true))
	gt.Log("Killed sub-process")

	require.NotNil(gt, capt.FindLog(
		testlog.NewMessageFilter("Sending interrupt")))

	require.NotNil(gt, capt.FindLog(
		testlog.NewMessageFilter("Sub-process stopped")))
}
