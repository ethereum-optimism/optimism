package devtest

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-service/testreq"
)

// P is used by the preset package and system backends as testing interface, to host package-wide resources.
type P interface {
	CommonT

	// WithCtx makes a copy of P with a specific context.
	// The ctx must match the test-scope of the existing context.
	// This function is used to create a P with annotated context, e.g. a specific resource.
	WithCtx(ctx context.Context) P

	// TempDir creates a temporary directory, and returns the file-path.
	// This directory is cleaned up at the end of the package,
	// and can be shared safely between tests that run in that package scope.
	TempDir() string

	// Cleanup runs the given function at the end of the package-scope.
	// This function will clean-up once the package-level testing is fully complete.
	// These resources can thus be shared safely between tests.
	Cleanup(fn func())

	// This distinguishes the interface from other testing interfaces,
	// such as the one used at test-level for test-scope resources.
	_PackageOnly()

	// Close closes the testing handle. This cancels the context and runs all cleanup.
	Close()
}

// implP is a P implementation that is used for package-level testing, and may be used by tooling as well.
// This is used in TestMain to manage resources that outlive a single test-scope.
type implP struct {
	// scopeName, for t.Name() purposes
	scopeName string

	// logger is used for logging. Regular test errors will also be redirected to get logged here.
	logger log.Logger

	// failNow will be called to register a failure.
	// The failure is intended to be critical if now==true.
	// The implementer can choose to panic, crit-log, exit, etc. as preferred.
	onFail func(now bool)

	ctx    context.Context
	cancel context.CancelFunc

	// cleanup stack
	cleanupLock    sync.Mutex
	cleanupBacklog []func()

	req *testreq.Assertions
}

var _ P = (*implP)(nil)

func (t *implP) Error(args ...any) {
	t.logger.Error(fmt.Sprintln(args...))
	t.Fail()
}

func (t *implP) Errorf(format string, args ...any) {
	t.logger.Error(fmt.Sprintf(format, args...))
	t.Fail()
}

func (t *implP) Fail() {
	t.onFail(false)
}

func (t *implP) FailNow() {
	t.onFail(true)
}

func (t *implP) TempDir() string {
	// The last "*" will be replaced with the random temp dir name
	tempDir, err := os.MkdirTemp("", "op-dev-*")
	if err != nil {
		t.Errorf("failed to create temp dir: %v", err)
		t.FailNow()
	}
	require.NotEmpty(t, tempDir, "sanity check temp-dir path is not empty")
	require.NotEqual(t, "/", tempDir, "sanity-check temp-dir is not root")
	t.Cleanup(func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.logger.Error("Failed to clean up temp dir", "dir", tempDir, "err", err)
		}
	})
	return tempDir
}

func (t *implP) Cleanup(fn func()) {
	t.cleanupLock.Lock()
	defer t.cleanupLock.Unlock()
	t.cleanupBacklog = append(t.cleanupBacklog, fn)
}

func (t *implP) Log(args ...any) {
	t.logger.Info(fmt.Sprintln(args...))
}

func (t *implP) Logf(format string, args ...any) {
	t.logger.Info(fmt.Sprintf(format, args...))
}

func (t *implP) Helper() {
	// no-op
}

func (t *implP) Name() string {
	return t.scopeName
}

func (t *implP) Logger() log.Logger {
	return t.logger
}

func (t *implP) Tracer() trace.Tracer {
	return otel.Tracer(t.Name())
}

func (t *implP) Ctx() context.Context {
	return t.ctx
}

type wrapP struct {
	ctx    context.Context
	logger log.Logger
	req    *testreq.Assertions
	P
}

var _ P = (*wrapP)(nil)

func (p *wrapP) Ctx() context.Context {
	return p.ctx
}

func (p *wrapP) Logger() log.Logger {
	return p.logger
}

func (t *implP) WithCtx(ctx context.Context) P {
	expected := TestScope(t.ctx)
	got := TestScope(ctx)
	t.req.Equal(expected, got, "cannot replace context with different test-scope")
	logger := t.logger.New()
	logger.SetContext(ctx)
	out := &wrapP{ctx: ctx, logger: logger, P: t}
	out.req = testreq.New(out)
	return out
}

func (t *implP) Require() *testreq.Assertions {
	return t.req
}

// Close runs the cleanup of this implP implementation.
//
// This cancels the package-wide test context.
//
// It then runs the backlog of cleanup functions, in reverse order (last registered cleanup runs first).
// It's inspired by the Go cleanup handler, fully cleaning up,
// even continuing to clean up when panics happen.
// It does not recover the go-routine from panicking however, that is up to the caller.
func (t *implP) Close() {
	// run remaining cleanups, even if a cleanup panics,
	// but don't recover the panic
	defer func() {
		t.cleanupLock.Lock()
		recur := len(t.cleanupBacklog) > 0
		t.cleanupLock.Unlock()
		if recur {
			t.logger.Error("Last cleanup panicked, continuing cleanup attempt now")
			t.Close()
		}
	}()

	for {
		// Pop a cleanup item, and execute it in unlocked state,
		// in case cleanups produce new cleanups.
		var cleanup func()
		t.cleanupLock.Lock()
		if len(t.cleanupBacklog) > 0 {
			last := len(t.cleanupBacklog) - 1
			cleanup = t.cleanupBacklog[last]
			t.cleanupBacklog = t.cleanupBacklog[:last]
		}
		t.cleanupLock.Unlock()
		if cleanup == nil {
			return
		}
		cleanup()
	}
}

func (t *implP) _PackageOnly() {
	panic("do not use - this method only forces the interface to be unique")
}

func NewP(ctx context.Context, logger log.Logger, onFail func(now bool)) P {
	ctx, cancel := context.WithCancel(ctx)
	out := &implP{
		scopeName: "pkg",
		logger:    logger,
		onFail:    onFail,
		ctx:       AddTestScope(ctx, "pkg"),
		cancel:    cancel,
	}
	out.req = testreq.New(out)
	return out
}
