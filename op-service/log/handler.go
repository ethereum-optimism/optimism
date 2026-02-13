package log

import (
	"fmt"
	"io"
	"log/slog"
	"math"
	"math/big"
	"reflect"
	"time"

	"github.com/holiman/uint256"

	elog "github.com/ethereum/go-ethereum/log"
)

const (
	timeFormatMs                 = "2006-01-02T15:04:05.000-0700"
	levelMaxVerbosity slog.Level = math.MinInt
)

type leveler struct{ minLevel slog.Level }

func (l *leveler) Level() slog.Level {
	return l.minLevel
}

func JSONMsHandler(wr io.Writer) slog.Handler {
	return JSONMsHandlerWithLevel(wr, levelMaxVerbosity)
}

func JSONMsHandlerWithLevel(wr io.Writer, level slog.Level) slog.Handler {
	return slog.NewJSONHandler(wr, &slog.HandlerOptions{
		ReplaceAttr: builtinReplaceJSONMs,
		Level:       &leveler{level},
	})
}

func LogfmtMsHandler(wr io.Writer) slog.Handler {
	return slog.NewTextHandler(wr, &slog.HandlerOptions{
		ReplaceAttr: builtinReplaceLogfmtMs,
	})
}

func LogfmtMsHandlerWithLevel(wr io.Writer, level slog.Level) slog.Handler {
	return slog.NewTextHandler(wr, &slog.HandlerOptions{
		ReplaceAttr: builtinReplaceLogfmtMs,
		Level:       &leveler{level},
	})
}

func builtinReplaceLogfmtMs(_ []string, attr slog.Attr) slog.Attr {
	return builtinReplaceMs(nil, attr, true)
}

func builtinReplaceJSONMs(_ []string, attr slog.Attr) slog.Attr {
	return builtinReplaceMs(nil, attr, false)
}

func builtinReplaceMs(_ []string, attr slog.Attr, logfmt bool) slog.Attr {
	switch attr.Key {
	case slog.TimeKey:
		if attr.Value.Kind() == slog.KindTime {
			if logfmt {
				return slog.String("t", attr.Value.Time().Format(timeFormatMs))
			} else {
				return slog.Attr{Key: "t", Value: attr.Value}
			}
		}
	case slog.LevelKey:
		if l, ok := attr.Value.Any().(slog.Level); ok {
			attr = slog.Any("lvl", elog.LevelString(l))
			return attr
		}
	}

	switch v := attr.Value.Any().(type) {
	case time.Time:
		if logfmt {
			attr = slog.String(attr.Key, v.Format(timeFormatMs))
		}
	case *big.Int:
		if v == nil {
			attr.Value = slog.StringValue("<nil>")
		} else {
			attr.Value = slog.StringValue(v.String())
		}
	case *uint256.Int:
		if v == nil {
			attr.Value = slog.StringValue("<nil>")
		} else {
			attr.Value = slog.StringValue(v.Dec())
		}
	case fmt.Stringer:
		if v == nil || (reflect.ValueOf(v).Kind() == reflect.Pointer && reflect.ValueOf(v).IsNil()) {
			attr.Value = slog.StringValue("<nil>")
		} else {
			attr.Value = slog.StringValue(v.String())
		}
	}
	return attr
}
