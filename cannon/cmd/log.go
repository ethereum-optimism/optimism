package cmd

import (
	"io"

	"golang.org/x/exp/slog"

	"github.com/ethereum/go-ethereum/log"
)

func Logger(w io.Writer, lvl slog.Level) log.Logger {
	return log.NewLogger(log.LogfmtHandlerWithLevel(w, lvl))
}
