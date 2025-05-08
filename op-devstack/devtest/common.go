package devtest

import (
	"context"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
)

// CommonT is a subset of testing.T, extended with a few common utils.
// This interface should not be used directly. Instead, use T in test-scope, or P when operating at package level.
//
// This CommonT interface is minimal enough such that it can be implemented by tooling,
// and a *testing.T can be used with minimal wrapping.
type CommonT interface {
	Errorf(format string, args ...interface{})
	FailNow()

	TempDir() string
	Cleanup(fn func())
	Logf(format string, args ...any)
	Helper()
	Name() string

	Logger() Logger
	Tracer() trace.Tracer
	Ctx() context.Context
	Require() *require.Assertions
}
