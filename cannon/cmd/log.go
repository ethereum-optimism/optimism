package cmd

import (
	"io"
	"log/slog"
	"os"

	"golang.org/x/term"

	"github.com/ethereum/go-ethereum/log"
)

func Logger(w io.Writer, lvl slog.Level) log.Logger {
	if term.IsTerminal(int(os.Stdout.Fd())) {
		return log.NewLogger(log.LogfmtHandlerWithLevel(w, lvl))
	} else {
		return log.NewLogger(rawLogHandler(w, lvl))
	}
}

// rawLogHandler returns a handler that strips out the time attribute
func rawLogHandler(wr io.Writer, lvl slog.Level) slog.Handler {
	return slog.NewTextHandler(wr, &slog.HandlerOptions{
		ReplaceAttr: replaceAttr,
		Level:       &leveler{lvl},
	})
}

type leveler struct{ minLevel slog.Level }

func (l *leveler) Level() slog.Level {
	return l.minLevel
}

func replaceAttr(_ []string, attr slog.Attr) slog.Attr {
	if attr.Key == slog.TimeKey {
		return slog.Attr{}
	}
	return attr
}
