package main

import (
	"os"
	"runtime"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-program/client"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
)

var _ = func() bool {
	// Disable mem profiling, to avoid a lot of unnecessary floating point ops
	// TODO(cp-903) Consider cutting this in favor patching envar "GODEBUG=memprofilerate=0" into cannon go programs
	runtime.MemProfileRate = 0
	return true
}()

func main() {
	// Default to a machine parsable but relatively human friendly log format.
	// Don't do anything fancy to detect if color output is supported.
	logger := oplog.NewLogger(os.Stdout, oplog.CLIConfig{
		Level:  log.LevelInfo,
		Format: oplog.FormatLogFmt,
		Color:  false,
	})
	oplog.SetGlobalLogHandler(logger.Handler())
	client.Main(logger)
}
