package devtest

import "github.com/stretchr/testify/require"

// gateAdapter implements the require.TestingT interface by using skips instead of errors.
// This is used to program test-gate conditions easily.
// The underlying test-interface may remap the test-skip to a test-error,
// if skips are not expected / allowed.
type gateAdapter struct {
	inner interface {
		Helper()
		Skipf(format string, args ...any)
		SkipNow()
	}
}

var _ require.TestingT = (*gateAdapter)(nil)

func (g *gateAdapter) Errorf(format string, args ...interface{}) {
	g.inner.Helper()
	g.inner.Skipf(format, args...)
}

func (g *gateAdapter) FailNow() {
	g.inner.Helper()
	g.inner.SkipNow()
}

func (g *gateAdapter) Helper() {
	g.inner.Helper()
}
