package op_e2e

import (
	"os"
	"testing"
)

var enableParallelTesting bool = os.Getenv("OP_E2E_DISABLE_PARALLEL") != "true"

func InitParallel(t *testing.T, opts ...func(t *testing.T)) {
	t.Helper()
	if enableParallelTesting {
		t.Parallel()
	}
	for _, opt := range opts {
		opt(t)
	}
}

func SkipIfHTTP(t *testing.T) {
	if UseHTTP() {
		t.Skip("Skipping test because HTTP connection is in use")
	}
}
