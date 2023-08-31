package op_e2e

import (
	"os"
	"testing"

	"github.com/ethereum-optimism/optimism/op-e2e/config"
	"github.com/ethereum/go-ethereum/log"
)

var enableParallelTesting bool = os.Getenv("OP_E2E_DISABLE_PARALLEL") != "true"

func InitParallel(t *testing.T) {
	t.Helper()
	if enableParallelTesting {
		t.Parallel()
	}
	if config.EthNodeVerbosity < 0 {
		log.Root().SetHandler(log.DiscardHandler())
	}
}
