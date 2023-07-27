package log

import (
	"sync"

	"github.com/ethereum/go-ethereum/log"
)

type Writer struct {
	log     func(str string, ctx ...interface{})
	lock    sync.Mutex
	pending []byte
}

func NewWriter(l log.Logger, lvl log.Lvl) *Writer {
	var logMethod func(str string, ctx ...interface{})
	switch lvl {
	case log.LvlTrace:
		logMethod = l.Trace
	case log.LvlDebug:
		logMethod = l.Debug
	case log.LvlInfo:
		logMethod = l.Info
	case log.LvlWarn:
		logMethod = l.Warn
	case log.LvlError:
		logMethod = l.Error
	case log.LvlCrit:
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
