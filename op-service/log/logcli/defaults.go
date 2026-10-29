package logcli

import (
	"os"

	"github.com/ethereum-optimism/optimism/op-service/log"
)

func SetupDefaults() {
	SetGlobalLogHandler(log.LogfmtHandlerWithLevel(os.Stdout, log.LevelInfo))
}
