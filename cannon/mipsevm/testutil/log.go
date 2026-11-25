package testutil

import (
	"os"

	"github.com/ethereum-optimism/optimism/op-service/log"
)

func CreateLogger() log.Logger {
	return log.NewLogger(log.LogfmtHandlerWithLevel(os.Stdout, log.LevelInfo))
}
