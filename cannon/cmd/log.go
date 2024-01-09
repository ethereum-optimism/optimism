package cmd

import (
	"io"

	"github.com/ethereum/go-ethereum/log"
	"golang.org/x/exp/slog"
)

func Logger(w io.Writer, lvl slog.Level) log.Logger {
	return log.NewLogger(log.LogfmtHandlerWithLevel(w, lvl))
}
