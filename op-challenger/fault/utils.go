package fault

import (
	"github.com/ethereum/go-ethereum/log"
)

type LogBuilder struct {
	ctx []interface{}
}

// NewLogBuilder creates a new [logBuilder] instance.
func NewLogBuilder() *LogBuilder {
	return &LogBuilder{}
}

// With adds a new [key, value] pair to the context.
func (l *LogBuilder) With(key string, value interface{}) {
	l.ctx = append(l.ctx, key, value)
}

// Build returns the new [log.Logger] with the built context.
func (l *LogBuilder) Build() log.Logger {
	return log.New(l.ctx...)
}
