package log

import (
	"os"

	"github.com/ethereum/go-ethereum/log"
)

func SetupDefaults() {
	SetGlobalLogHandler(log.LogfmtHandlerWithLevel(os.Stdout, log.LevelInfo))
}
