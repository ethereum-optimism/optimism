package sysgo

import (
	"context"
	"io"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-service/log"
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

	require.NoError(gt, sp.Start("/bin/sh", []string{"-c", "trap '' INT; exec /bin/sleep 3600"}, []string{}))

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

func newTestSubProcess(gt *testing.T) (*SubProcess, *testlog.CapturingHandler) {
	tLog := testlog.Logger(gt, log.LevelInfo)
	logger, capt := testlog.CaptureLogger(gt, log.LevelInfo)
	p := devtest.NewP(context.Background(), logger, func(bool) { panic("fail") }, func() { panic("skip") })
	gt.Cleanup(p.Close)
	logCallback := logpipe.LogCallback(func(line []byte) {
		logger.Info(string(line))
		tLog.Info("Sub-process logged message", "line", string(line))
	})
	return NewSubProcess(p, logCallback, logCallback), capt
}

func TestSubProcessKillEndsProcessThatIgnoresInterrupts(gt *testing.T) {
	sp, capt := newTestSubProcess(gt)
	require.NoError(gt, sp.Start("/bin/sh", []string{"-c", "trap '' INT TERM; exec /bin/sleep 3600"}, []string{}))

	require.NoError(gt, sp.Kill())

	require.Nil(gt, capt.FindLog(testlog.NewMessageFilter("Sending interrupt")), "Kill must not interrupt first")
	require.NotNil(gt, capt.FindLog(testlog.NewMessageFilter("Sub-process killed")))

	require.NoError(gt, sp.Start("/bin/echo", []string{"restart ok"}, []string{}), "the sub-process must restart after a kill")
	require.NoError(gt, sp.Stop(false))
	require.NotNil(gt, capt.FindLog(testlog.NewMessageFilter("restart ok")))
}

func TestSubProcessKillOfExitedProcess(gt *testing.T) {
	sp, capt := newTestSubProcess(gt)
	require.NoError(gt, sp.Start("/bin/echo", []string{"done"}, []string{}))
	<-sp.Exited()

	require.ErrorContains(gt, sp.Kill(), "exited with status 0 before it could be killed", "an exit of its own must fail the kill")
	require.NotNil(gt, capt.FindLog(testlog.NewMessageFilter("Sub-process exited before it could be killed")))
	require.Nil(gt, capt.FindLog(testlog.NewMessageFilter("Sub-process killed")), "an exit of its own must not read as a kill")
	require.NoError(gt, sp.Kill(), "a second kill is a no-op")
}

func TestSubProcessKillOfCrashedProcess(gt *testing.T) {
	sp, _ := newTestSubProcess(gt)
	require.NoError(gt, sp.Start("/bin/sh", []string{"-c", "exit 3"}, []string{}))
	<-sp.Exited()

	err := sp.Kill()
	var exitErr *exec.ExitError
	require.ErrorAs(gt, err, &exitErr, "the kill must report the exit it found")
	require.Equal(gt, 3, exitErr.ExitCode())
	require.NoError(gt, sp.Start("/bin/echo", []string{"restart ok"}, []string{}), "the sub-process must restart after a failed kill")
	require.NoError(gt, sp.Stop(false))
}

func TestSubProcessKillOfProcessThatExitedBeforeItWasReaped(gt *testing.T) {
	sp, capt := newTestSubProcess(gt)
	childExitedReader, childStdout, err := os.Pipe()
	require.NoError(gt, err)
	defer childExitedReader.Close()
	cmd := exec.Command("/bin/sh", "-c", "exit 3")
	cmd.Stdout = childStdout
	require.NoError(gt, cmd.Start())
	require.NoError(gt, childStdout.Close())
	// The pipe reaches EOF once the child has exited, while it stays unreaped until reap closes.
	_, err = io.ReadAll(childExitedReader)
	require.NoError(gt, err)

	reap := make(chan struct{})
	sp.cmd, sp.exited = cmd, make(chan struct{})
	go func() {
		<-reap
		sp.waitErr = cmd.Wait()
		close(sp.exited)
	}()
	killErrCh := make(chan error, 1)
	go func() { killErrCh <- sp.Kill() }()
	require.Eventually(gt, func() bool {
		return capt.FindLog(testlog.NewMessageFilter("Killing sub-process")) != nil
	}, 5*time.Second, 10*time.Millisecond, "the kill must reach the unreaped process")
	close(reap)

	killErr := <-killErrCh
	var exitErr *exec.ExitError
	require.ErrorAs(gt, killErr, &exitErr, "the kill must report the exit that preceded it")
	require.Equal(gt, 3, exitErr.ExitCode())
	require.Nil(gt, capt.FindLog(testlog.NewMessageFilter("Sub-process killed")), "an exit of its own must not read as a kill")
}

func TestTerminatedBySIGKILL(gt *testing.T) {
	for _, testCase := range []struct {
		name   string
		script string
		want   bool
	}{
		{name: "exit_0", script: "exit 0", want: false},
		{name: "exit_3", script: "exit 3", want: false},
		{name: "sigkill", script: "kill -KILL $$", want: true},
		{name: "sigterm", script: "kill -TERM $$", want: false},
	} {
		gt.Run(testCase.name, func(gt *testing.T) {
			waitErr := exec.Command("/bin/sh", "-c", testCase.script).Run()
			require.Equal(gt, testCase.want, terminatedBySIGKILL(waitErr), "wait result: %v", waitErr)
		})
	}
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
