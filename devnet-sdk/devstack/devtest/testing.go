package devtest

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

const ExpectPreconditionsMet = "DEVNET_EXPECT_PRECONDITIONS_MET"

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
	Gate() *require.Assertions

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
	ctx    context.Context
	req    *require.Assertions
	gate   *require.Assertions
}

func mustNotSkip() bool {
	v := os.Getenv(ExpectPreconditionsMet)
	out, _ := strconv.ParseBool(v) // default to false
	return out
}

func (t *testingT) Errorf(format string, args ...interface{}) {
	t.t.Helper()
	t.t.Errorf(format, args...)
}

func (t *testingT) FailNow() {
	t.t.Helper()
	t.t.FailNow()
}

func (t *testingT) TempDir() string {
	return t.t.TempDir()
}

func (t *testingT) Cleanup(fn func()) {
	t.t.Cleanup(fn)
}

func (t *testingT) Logf(format string, args ...any) {
	t.t.Helper()
	// Note: we do not use t.Log directly, to keep the log-formatting more consistent
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

func (t *testingT) Ctx() context.Context {
	return t.ctx
}

func (t *testingT) Require() *require.Assertions {
	return t.req
}

func (t *testingT) Run(name string, fn func(T)) {
	t.t.Run(name, func(subGoT *testing.T) {
		ctx, cancel := context.WithCancel(t.ctx)
		subGoT.Cleanup(cancel)
		subT := &testingT{
			t:      subGoT,
			logger: t.logger.New("subtest", name),
			ctx:    ctx,
		}
		subT.req = require.New(subT)
		subT.gate = require.New(&gateAdapter{subT})
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
		t.t.Error(args...)
		return
	}
	t.t.Skip(args...)
}

func (t *testingT) Skipped() bool {
	t.t.Helper()
	return t.t.Skipped()
}

func (t *testingT) Skipf(format string, args ...any) {
	t.t.Helper()
	if mustNotSkip() {
		t.t.Errorf(format, args...)
		return
	}
	t.t.Skipf(format, args...)
}

func (t *testingT) SkipNow() {
	t.t.Helper()
	if mustNotSkip() {
		t.t.FailNow()
		return
	}
	t.t.SkipNow()
}

func (t *testingT) Gate() *require.Assertions {
	return t.gate
}

func (t *testingT) _TestOnly() {
	panic("do not use - this method only forces the interface to be unique")
}

var _ T = (*testingT)(nil)

// SerialT wraps around a test-logger and turns it into a T for devstack testing.
func SerialT(t *testing.T) T {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	logger := testlog.Logger(t, log.LevelInfo)
	out := &testingT{
		t:      t,
		logger: logger,
		ctx:    ctx,
	}
	out.req = require.New(out)
	out.gate = require.New(&gateAdapter{out})
	return out
}

// ParallelT creates a T interface with parallel testing enabled by default
func ParallelT(t *testing.T) T {
	out := SerialT(t)
	out.Parallel()
	return out
}
