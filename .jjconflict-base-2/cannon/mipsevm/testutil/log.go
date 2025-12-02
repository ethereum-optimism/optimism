package testutil

import (
	"os"

	"github.com/ethereum/go-ethereum/log"
)

func CreateLogger() log.Logger {
	return log.NewLogger(log.LogfmtHandlerWithLevel(os.Stdout, log.LevelInfo))
}
