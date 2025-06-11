package logfilter

import (
	"context"
	"log/slog"

	"github.com/ethereum/go-ethereum/log"
)

// LogFilter is a function that can adjust how a record is filtered.
// The ctx of the record logging call is provided to adapt to the call.
// The currently considered level of the logging is provided as input.
// The filter may return the same (to not affect the outcome), or adjust it up or down.
type LogFilter func(ctx context.Context, lvl slog.Level) slog.Level

func (fn LogFilter) Add(next LogFilter) LogFilter {
	return func(ctx context.Context, lvl slog.Level) slog.Level {
		return next(ctx, fn(ctx, lvl))
	}
}

// Combine applies all filters, from left to right.
func Combine(filters ...LogFilter) LogFilter {
	return func(ctx context.Context, lvl slog.Level) slog.Level {
		for _, fn := range filters {
			lvl = fn(ctx, lvl)
		}
		return lvl
	}
}

// Noop does not apply any change to the log-level
func Noop() LogFilter {
	return func(ctx context.Context, lvl slog.Level) slog.Level {
		return lvl
	}
}

// MuteLvl is a special log level, that we always consider muted.
const MuteLvl = log.LevelTrace - 10

// Mute will filter everything
func Mute() LogFilter {
	return func(_ context.Context, lvl slog.Level) slog.Level {
		return MuteLvl
	}
}

// As will consider everything to be the given replacement log level
func As(replacement slog.Level) LogFilter {
	return func(_ context.Context, lvl slog.Level) slog.Level {
		return replacement
	}
}

// Add will add a log level delta, to adjust up or down
func Add(delta slog.Level) LogFilter {
	return func(ctx context.Context, lvl slog.Level) slog.Level {
		return lvl + delta
	}
}

// Minimum will mute the log if the minimum is not met
func Minimum(minLvl slog.Level) LogFilter {
	return func(ctx context.Context, lvl slog.Level) slog.Level {
		if lvl < minLvl {
			return MuteLvl
		}
		return lvl
	}
}

// IfMatch checks for a given key/value pair in the context, and applies the given next filter if found.
func IfMatch[K comparable, V comparable](k K, v V, next LogFilter) LogFilter {
	return func(ctx context.Context, lvl slog.Level) slog.Level {
		x := ctx.Value(k)
		if x == nil {
			return lvl
		}
		res, ok := x.(V)
		if !ok || res != v {
			return lvl
		}
		return next(ctx, lvl)
	}
}
