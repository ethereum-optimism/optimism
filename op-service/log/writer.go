package log

import (
	"log/slog"
	"sync"
)

type Writer struct {
	log     func(str string, ctx ...any)
	lock    sync.Mutex
	pending []byte
}

func NewWriter(l Logger, lvl slog.Level) *Writer {
	var logMethod func(str string, ctx ...any)
	switch lvl {
	case LevelTrace:
		logMethod = l.Trace
	case LevelDebug:
		logMethod = l.Debug
	case LevelInfo:
		logMethod = l.Info
	case LevelWarn:
		logMethod = l.Warn
	case LevelError:
		logMethod = l.Error
	case LevelCrit:
		logMethod = l.Crit
	default:
		// Cast lvl to int to avoid trying to convert it to a string which will fail for unknown types
		l.Error("Unknown log level. Using Error", "lvl", int(lvl))
		logMethod = l.Error
	}
	return &Writer{
		log: logMethod,
	}
}

func (w *Writer) Write(b []byte) (int, error) {
	w.lock.Lock()
	defer w.lock.Unlock()
	for _, c := range b {
		if c == '\n' {
			w.log(string(w.pending))
			w.pending = nil
			continue
		}
		w.pending = append(w.pending, c)
	}
	return len(b), nil
}

func (w *Writer) Close() error {
	w.lock.Lock()
	defer w.lock.Unlock()
	if len(w.pending) > 0 {
		w.log(string(w.pending))
		w.pending = nil
	}
	return nil
}
