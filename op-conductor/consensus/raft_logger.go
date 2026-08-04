package consensus

import (
	"io"

	"github.com/ethereum/go-ethereum/log"
	hclog "github.com/hashicorp/go-hclog"
)

type raftLogSink struct {
	log log.Logger
}

func (s *raftLogSink) Accept(name string, level hclog.Level, msg string, args ...interface{}) {
	if name != "" {
		args = append([]interface{}{"component", name}, args...)
	}

	switch level {
	case hclog.Trace:
		s.log.Trace(msg, args...)
	case hclog.Debug:
		s.log.Debug(msg, args...)
	case hclog.Info, hclog.NoLevel:
		s.log.Info(msg, args...)
	case hclog.Warn:
		s.log.Warn(msg, args...)
	case hclog.Error:
		s.log.Error(msg, args...)
	}
}

func newRaftLogger(log log.Logger) hclog.Logger {
	logger := hclog.NewInterceptLogger(&hclog.LoggerOptions{
		Name:   "raft",
		Level:  hclog.Trace,
		Output: io.Discard,
	})
	logger.RegisterSink(&raftLogSink{log: log})
	return logger
}
