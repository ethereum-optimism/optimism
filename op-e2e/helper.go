package op_e2e

import (
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-e2e/config"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

var enableParallelTesting bool = os.Getenv("OP_E2E_DISABLE_PARALLEL") != "true"

func InitParallel(t *testing.T) {
	t.Helper()
	if enableParallelTesting {
		t.Parallel()
	}
	lvl := log.Lvl(config.EthNodeVerbosity)
	if lvl < log.LvlCrit {
		log.Root().SetHandler(log.DiscardHandler())
	} else if lvl > log.LvlTrace { // clip to trace level
		lvl = log.LvlTrace
	}
	h := testlog.Handler(t, lvl, log.TerminalFormat(false)) // some CI logs do not handle colors well
	oplog.SetGlobalLogHandler(h)
}
