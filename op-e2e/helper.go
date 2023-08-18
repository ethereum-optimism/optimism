package op_e2e

import (
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/log"
)

var verboseEthNodes bool
var externalL2Nodes string

var enableParallelTesting bool = os.Getenv("OP_E2E_DISABLE_PARALLEL") != "true"

func InitParallel(t *testing.T) {
	t.Helper()
	if enableParallelTesting {
		t.Parallel()
	}
	if !verboseEthNodes {
		log.Root().SetHandler(log.DiscardHandler())
	}
}
