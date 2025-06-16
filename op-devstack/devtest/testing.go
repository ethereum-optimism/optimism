package devtest

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/devnet-sdk/telemetry"
	"github.com/ethereum-optimism/optimism/op-service/log/logfilter"
	"github.com/ethereum-optimism/optimism/op-service/logmods"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testreq"
)

const ExpectPreconditionsMet = "DEVNET_EXPECT_PRECONDITIONS_MET"

var (
	// RootContext is the context that is used for the root of the test suite.
	// It should be set for good before any tests are run.
	RootContext = context.Background()
)

type T interface {
	CommonT

	// TempDir creates a temporary directory, and returns the file-path.
	// This directory is cleaned up at the end of the test, and must not be shared between tests.
	TempDir() string

	// Cleanup runs the given function at the end of the test-scope,
	// or at the end of the sub-test (if this is a nested test).
	// This function will clean-up before the package-level testing scope may be complete.
	// Do not use the test-scope cleanup with shared resources.
	Cleanup(fn func())

	// Run runs the given function in as a sub-test.
	Run(name string, fn func(T))

	// Ctx returns a context that will be canceled at the end of this (sub) test-scope,
	// and inherits the context of the parent-test-scope.
	Ctx() context.Context

	// WithCtx makes a copy of T with a specific context.
	// The ctx must match the test-scope of the existing context.
	// This function is used to create a T with annotated context, e.g. a specific resource, rather than a sub-scope.
	WithCtx(ctx context.Context) T

	// Parallel signals that this test is to be run in parallel with (and only with) other parallel tests.
	Parallel()

	// Skip is equivalent to Log followed by SkipNow.
	Skip(args ...any)
	// Skipped reports whether the test was skipped.
	Skipped() bool
	// Skipf is equivalent to Logf followed by SkipNow.
	Skipf(format string, args ...any)
	// SkipNow marks the test as skipped and stops test execution.
	// It is remapped to FailNow if the env var DEVNET_EXPECT_PRECONDITIONS_MET is set to true.
	SkipNow()

	// Gate provides everything that Require does, but skips instead of fails the test upon error.
	Gate() *testreq.Assertions

	// Deadline reports the time at which the test binary will have
	// exceeded the timeout specified by the -timeout flag.
	//
	// The ok result is false if the -timeout flag indicates "no timeout" (0).
	Deadline() (deadline time.Time, ok bool)

	// This distinguishes the interface from other testing interfaces,
	// such as the one used at package-level for shared system construction.
	_TestOnly()
}

// This testing subset is sufficient for the require.Assertions to work.
var _ require.TestingT = T(nil)

// testingT implements the T interface by wrapping around a regular golang testing.T
type testingT struct {
	t      *testing.T
	logger log.Logger
	tracer trace.Tracer
	ctx    context.Context
	req    *testreq.Assertions
	gate   *testreq.Assertions
}

func mustNotSkip() bool {
	v := os.Getenv(ExpectPreconditionsMet)
	out, _ := strconv.ParseBool(v) // default to false
	return out
}

func (t *testingT) Error(args ...any) {
	t.t.Helper()
	// Note: the test-logger catches panics when the test is logged to after test-end.
	// Note: we do not use t.Error directly, to keep the log-formatting more consistent.
	t.logger.Error(fmt.Sprintln(args...))
	t.Fail()
}

func (t *testingT) Errorf(format string, args ...any) {
	t.t.Helper()
	// Note: the test-logger catches panics when the test is logged to after test-end.
	// Note: we do not use t.Errorf directly, to keep the log-formatting more consistent.
	t.logger.Error(fmt.Sprintf(format, args...))
	t.Fail()
}

func (t *testingT) Fail() {
	t.t.Helper()
	// if we already closed and failed, then this error is stale
	if t.ctx.Err() != nil && t.t.Failed() {
		return
	}
	t.t.Fail()
}

func (t *testingT) FailNow() {
	t.t.Helper()
	// If we already closed and failed the test-scope, then there is nothing to do.
	// This happens on e.g. a go-routine spawned by require.Eventually, when the time runs out,
	// the ctx is closed, a shared resource fails to do a lookup because of the ctx-timeout,
	// and the eventually-condition then does a no-error check, causing the test-scope to error after it already had.
	if t.ctx.Err() != nil && t.t.Failed() {
		// Exit the go-routine that is running us (actual testing.T FailNow does this too).
		// Still runs deferred calls on this go-routine.
		runtime.Goexit()
		return
	}
	t.t.FailNow()
}

func (t *testingT) TempDir() string {
	return t.t.TempDir()
}

func (t *testingT) Cleanup(fn func()) {
	t.t.Cleanup(fn)
}

func (t *testingT) Log(args ...any) {
	t.t.Helper()
	// Note: the test-logger catches panics when the test is logged to after test-end.
	// Note: we do not use t.Log directly, to keep the log-formatting more consistent.
	t.logger.Info(fmt.Sprintln(args...))
}

func (t *testingT) Logf(format string, args ...any) {
	t.t.Helper()
	// Note: the test-logger catches panics when the test is logged to after test-end.
	// Note: we do not use t.Logf directly, to keep the log-formatting more consistent.
	t.logger.Info(fmt.Sprintf(format, args...))
}

func (t *testingT) Helper() {
	t.t.Helper()
}

func (t *testingT) Name() string {
	return t.t.Name()
}

func (t *testingT) Logger() log.Logger {
	return t.logger
}

func (t *testingT) Tracer() trace.Tracer {
	return t.tracer
}

func (t *testingT) Ctx() context.Context {
	return t.ctx
}

func (t *testingT) WithCtx(ctx context.Context) T {
	expected := TestScope(t.ctx)
	got := TestScope(ctx)
	t.req.Equal(expected, got, "cannot replace context with different test-scope")
	logger := t.logger.New()
	logger.SetContext(ctx)
	out := &testingT{
		t:      t.t,
		logger: logger,
		tracer: t.tracer,
		ctx:    ctx,
	}
	out.req = testreq.New(out)
	out.gate = testreq.New(&gateAdapter{out})
	return out
}

func (t *testingT) Require() *testreq.Assertions {
	return t.req
}

func (t *testingT) Run(name string, fn func(T)) {
	baseName := t.Name()
	t.t.Run(name, func(subGoT *testing.T) {
		ctx := AddTestScope(t.ctx, name)

		ctx, cancel := context.WithCancel(ctx)
		subGoT.Cleanup(cancel)

		tracer := otel.Tracer(baseName + "::" + name)
		ctx, span := tracer.Start(ctx, name)
		subGoT.Cleanup(func() {
			span.End()
		})
		logger := t.logger.New()
		logger.SetContext(ctx) // attach the sub-test context as default log-context

		subT := &testingT{
			t:      subGoT,
			logger: logger,
			tracer: tracer,
			ctx:    ctx,
		}
		subT.req = testreq.New(subT)
		subT.gate = testreq.New(&gateAdapter{subT})
		fn(subT)
	})
}

func (t *testingT) Parallel() {
	t.logger.Info("Running test in parallel")
	t.t.Parallel()
}

func (t *testingT) Skip(args ...any) {
	t.t.Helper()
	if mustNotSkip() {
		t.Error("Unexpected test-skip!", fmt.Sprintln(args...))
		return
	}
	t.Log(args...)
	t.t.SkipNow()
}

func (t *testingT) Skipped() bool {
	t.t.Helper()
	return t.t.Skipped()
}

func (t *testingT) Skipf(format string, args ...any) {
	t.t.Helper()
	if mustNotSkip() {
		t.Error("Unexpected test-skip!", fmt.Sprintf(format, args...))
		return
	}
	t.Logf(format, args...)
	t.t.SkipNow()
}

func (t *testingT) SkipNow() {
	t.t.Helper()
	if mustNotSkip() {
		t.FailNow()
		return
	}
	t.t.SkipNow()
}

func (t *testingT) Gate() *testreq.Assertions {
	return t.gate
}

// Deadline reports the time at which the test binary will have
// exceeded the timeout specified by the -timeout flag.
//
// The ok result is false if the -timeout flag indicates "no timeout" (0).
func (t *testingT) Deadline() (deadline time.Time, ok bool) {
	return t.t.Deadline()
}

func (t *testingT) _TestOnly() {
	panic("do not use - this method only forces the interface to be unique")
}

var _ T = (*testingT)(nil)

// DefaultTestLogLevel is set to info level to show relevant logs without being overly verbose unless configured otherwise.
var DefaultTestLogLevel = log.LevelInfo

// SerialT wraps around a test-logger and turns it into a T for devstack testing.
func SerialT(t *testing.T) T {
	ctx := RootContext
	ctx = AddTestScope(ctx, t.Name())

	var cancel context.CancelFunc
	if deadline, hasDeadline := t.Deadline(); hasDeadline {
		ctx, cancel = context.WithDeadline(ctx, deadline.Add(-3*time.Second))
	} else {
		ctx, cancel = context.WithCancel(ctx)
	}
	t.Cleanup(cancel)

	tracer := otel.Tracer(t.Name())
	ctx, span := tracer.Start(ctx, t.Name())
	t.Cleanup(func() {
		span.End()
	})

	// Set the lowest default log-level, so the log-filters on top can apply correctly
	logger := testlog.LoggerWithHandlerMod(t, log.LevelTrace,
		telemetry.WrapHandler, logfilter.WrapFilterHandler, logfilter.WrapContextHandler)
	h, ok := logmods.FindHandler[logfilter.FilterHandler](logger.Handler())
	if ok {
		// Apply default log level. This may be overridden later.
		h.Set(logfilter.DefaultMute(logfilter.Level(log.LevelInfo).Show()))
	}
	logger.SetContext(ctx) // Set the default context; any log call without context will use this

	out := &testingT{
		t:      t,
		logger: logger,
		tracer: tracer,
		ctx:    ctx,
	}
	out.req = testreq.New(out)
	out.gate = testreq.New(&gateAdapter{out})
	return out
}

// ParallelT creates a T interface with parallel testing enabled by default
func ParallelT(t *testing.T) T {
	out := SerialT(t)
	out.Parallel()
	return out
}
