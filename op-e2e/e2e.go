package op_e2e

import (
	"os"
	"testing"

	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils"
)

func RunMain(m *testing.M) {
	os.Exit(m.Run())
}

var enableParallelTesting bool = os.Getenv("OP_E2E_DISABLE_PARALLEL") != "true"

func InitParallel(t e2eutils.TestingBase, args ...func(t e2eutils.TestingBase)) {
	t.Helper()
	if enableParallelTesting {
		t.Parallel()
	}
	for _, arg := range args {
		arg(t)
	}
}

func UsesCannon(t e2eutils.TestingBase) {
	if os.Getenv("OP_E2E_CANNON_ENABLED") == "false" {
		t.Skip("Skipping cannon test")
	}
}
