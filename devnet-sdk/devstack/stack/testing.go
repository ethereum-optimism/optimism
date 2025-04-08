package stack

import (
	"fmt"
	"os"
	"sync"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/log"
)

// T is a minimal subset of testing.T
// This can be implemented by tooling, or a *testing.T can be used directly.
// The T interface is only used in the stack package for sanity-checks,
// where local failure there and then is preferable over bubbling up the error.
type T interface {
	Errorf(format string, args ...interface{})
	FailNow()
	TempDir() string
	Cleanup(fn func())
	Logf(format string, args ...any)
	Helper()
	Name() string
}

// This testing subset is sufficient for the require.Assertions to work.
var _ require.TestingT = T(nil)

// ToolingT is a T implementation that can be used in tooling,
// when the devnet-SDK is not used in a regular Go test.
type ToolingT struct {
	// TestName, for t.Name() purposes
	TestName string

	// Errors will be logged here.
	Log log.Logger

	// Fail will be called to register a critical failure.
	// The implementer can choose to panic, crit-log, exit, etc. as preferred.
	Fail func()

	// cleanup stack
	cleanupLock    sync.Mutex
	cleanupBacklog []func()
}

var _ T = (*ToolingT)(nil)

func (t *ToolingT) Errorf(format string, args ...interface{}) {
	t.Log.Error(fmt.Sprintf(format, args...))
}

func (t *ToolingT) FailNow() {
	t.Fail()
}

func (t *ToolingT) TempDir() string {
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
			t.Log.Error("Failed to clean up temp dir", "dir", tempDir, "err", err)
		}
	})
	return tempDir
}

func (t *ToolingT) Cleanup(fn func()) {
	t.cleanupLock.Lock()
	defer t.cleanupLock.Unlock()
	t.cleanupBacklog = append(t.cleanupBacklog, fn)
}

// RunCleanup runs the backlog of cleanup functions.
// It's inspired by the Go cleanup handler, fully cleaning up,
// even continuing to clean up when panics happen.
// It does not recover the go-routine from panicking however, that is up to the caller.
func (t *ToolingT) RunCleanup() {
	// run remaining cleanups, even if a cleanup panics,
	// but don't recover the panic
	defer func() {
		t.cleanupLock.Lock()
		recur := len(t.cleanupBacklog) > 0
		t.cleanupLock.Unlock()
		if recur {
			t.Log.Error("Last clean panicked, continuing cleanup attempt now")
			t.RunCleanup()
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

func (t *ToolingT) Logf(format string, args ...any) {
	t.Log.Info(fmt.Sprintf(format, args...))
}

func (t *ToolingT) Helper() {
	// no-op
}

func (t *ToolingT) Name() string {
	return t.TestName
}

func NewToolingT(name string, logger log.Logger) *ToolingT {
	t := &ToolingT{
		TestName: name,
		Log:      logger,
		Fail: func() {
			logger.Error("Exiting now...")
			os.Exit(1)
		},
	}
	return t
}
