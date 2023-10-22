package database

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/log"

	"gorm.io/gorm/logger"
)

var (
	_ logger.Interface = Logger{}

	SlowThresholdMilliseconds int64 = 500
)

type Logger struct {
	log log.Logger
}

func newLogger(log log.Logger) Logger {
	return Logger{log}
}

func (l Logger) LogMode(lvl logger.LogLevel) logger.Interface {
	return l
}

func (l Logger) Info(ctx context.Context, msg string, data ...interface{}) {
	l.log.Info(fmt.Sprintf(msg, data...))
}

func (l Logger) Warn(ctx context.Context, msg string, data ...interface{}) {
	l.log.Warn(fmt.Sprintf(msg, data...))
}

func (l Logger) Error(ctx context.Context, msg string, data ...interface{}) {
	l.log.Error(fmt.Sprintf(msg, data...))
}

func (l Logger) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	elapsedMs := time.Since(begin).Milliseconds()

	// omit any values for batch inserts as they can be very long
	sql, rows := fc()
	if i := strings.Index(strings.ToLower(sql), "values"); i > 0 {
		sql = fmt.Sprintf("%sVALUES (...)", sql[:i])
	}

	if elapsedMs < SlowThresholdMilliseconds {
		l.log.Debug("database operation", "duration_ms", elapsedMs, "rows_affected", rows, "sql", sql)
	} else {
		l.log.Warn("database operation", "duration_ms", elapsedMs, "rows_affected", rows, "sql", sql)
	}
}
