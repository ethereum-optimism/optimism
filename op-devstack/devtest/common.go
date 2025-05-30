package devtest

import (
	"context"

	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum/go-ethereum/log"

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

	Logger() log.Logger
	Tracer() trace.Tracer
	Ctx() context.Context
	Require() *require.Assertions
}

type testScopeCtxKeyType struct{}

// testScopeCtxKey is a key added to the test-context to identify the test-scope.
var testScopeCtxKey = testScopeCtxKeyType{}

// TestScope retrieves the test-scope from the context
func TestScope(ctx context.Context) string {
	scope := ctx.Value(testScopeCtxKey)
	if scope == nil {
		return ""
	}
	return scope.(string)
}

// AddTestScope combines the sub-scope with the test-scope of the context,
// and returns a context with the updated scope value.
func AddTestScope(ctx context.Context, scope string) context.Context {
	prev := TestScope(ctx)
	ctx = oplog.RegisterLogAttrOnContext(ctx, "scope", testScopeCtxKey)
	return context.WithValue(ctx, testScopeCtxKey, prev+"/"+scope)
}
