package e2eutils

import (
	"context"
	"testing"
	"time"
)

// TestingBase is an interface used for standard Go testing.
// This interface is used for unit tests, benchmarks, and fuzz tests and also emulated in Hive.
//
// The Go testing.TB interface does not allow extensions by embedding the interface, so we repeat it here.
type TestingBase interface {
	Cleanup(func())
	Error(args ...any)
	Errorf(format string, args ...any)
	Fail()
	FailNow()
	Failed() bool
	Fatal(args ...any)
	Fatalf(format string, args ...any)
	Helper()
	Log(args ...any)
	Logf(format string, args ...any)
	Name() string
	Setenv(key, value string)
	Skip(args ...any)
	SkipNow()
	Skipf(format string, args ...any)
	Skipped() bool
	TempDir() string
	Parallel()
}

func TimeoutCtx(t *testing.T, timeout time.Duration) context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	return ctx
}
