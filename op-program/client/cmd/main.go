package main

import (
	"os"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-program/client"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
)

func main() {
	// Default to a machine parsable but relatively human friendly log format.
	// Don't do anything fancy to detect if color output is supported.
	logger := oplog.NewLogger(os.Stdout, oplog.CLIConfig{
		Level:  log.LvlInfo,
		Format: oplog.FormatLogFmt,
		Color:  false,
	})
	oplog.SetGlobalLogHandler(logger.GetHandler())
	client.Main(logger)
}
