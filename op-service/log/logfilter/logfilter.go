/*
Package logfilter provides a declarative log filtering framework for structured
logging.

This package uses a tri-state logic system (True, False, Unknown) for
sophisticated log filtering capabilities. It allows creation of complex
filtering expressions using logical operations (And, Or, Not) and provides both
level-based and attribute-based filtering mechanisms.

This implementation is not optimized for production use. The current
implementation prioritizes flexibility and ease of use over performance. In
high-throughput logging scenarios, the overhead of context manipulation, dynamic
attribute loading, and tri-state logic evaluation may significantly impact
application performance.

Core Concepts:

- FilterHandler: A slog.Handler wrapper that applies filtering rules
dynamically, supporting attribute inheritance and context-based matching.

- Filter: A top-level predicate function that integrates with the log handler.
The two built-in filters offered are `logfilter.ShowDefault(<statements>...)`
and `logfilter.HideDefault(<statements>...)`, which integrate deeply with the
statement and selector concepts outlined next.

- Selector: A function describing a set of logs, returning a Tri (true, false or
undefined). A selector should return true if the log is within the selector set,
false if log is not within the selector set, and undefined if the domain of the
selector does not apply to the log. For example, a selector on chainID would
return undefined if the log does not contain a chainID because the log most
likely originated at a higher level in logger chain which lacks a chainID
context kv pair.

- Statements: Selectors must be combined with a designator (`.Show()` or
`.Mute()`) to form statements which can be fed to either
`logfilter.DefaultShow()` or `logfilter.DefaultMute()`.

Example usage:

	// Create a filter that shows debug logs for specific components
	filter := logfilter.DefaultShow(
		logfilter.Level(slog.LevelInfo).Show(),                    // Show info and above by default
		logfilter.Select("component", "database").And(        // For database component, also show debug logs
			Level(slog.LevelDebug)).Show(),
	)

	// Apply to a handler
	handler := logfilter.WrapFilterHandler(originalHandler)
	handler.Set(filter)
	logger := log.New(handler) // `log` references the go-ethereum log package

	logger.Info("general operation") // shown
	logger.Debug("debug operation") // muted

	ctx := context.Background()
	ctx = logfilter.AddLogAttrToContext(ctx, "component", slog.String("database"))
	databaseLogger := logger.SetContext(ctx)

	databaseLogger.Info("database operation") // shown
	databaseLogger.Debug("debug operation") // shown
*/
package logfilter

import (
	"context"
	"log/slog"
	"sync/atomic"

	"github.com/ethereum-optimism/optimism/op-service/logmods"
	"github.com/ethereum-optimism/optimism/op-service/tri"
)

// ============================================================================
// Declarative Filter Implementation Types
// ============================================================================

// Selector is an expression that can determine whether a log record should be included.
// The ctx of the record logging call is provided to adapt to the call.
// The currently considered level of the logging is provided as input.
// The filter returns true if the log should be included, false if it should be filtered out.
type Selector func(ctx context.Context, lvl slog.Level) tri.Tri

func (f Selector) And(other Selector) Selector {
	return func(ctx context.Context, lvl slog.Level) tri.Tri {
		return f(ctx, lvl).And(other(ctx, lvl))
	}
}

func (f Selector) Or(other Selector) Selector {
	return func(ctx context.Context, lvl slog.Level) tri.Tri {
		return f(ctx, lvl).Or(other(ctx, lvl))
	}
}

func (f Selector) Not() Selector {
	return func(ctx context.Context, lvl slog.Level) tri.Tri {
		return f(ctx, lvl).Not()
	}
}

// Level returns a LogFilter that shows logs at or above the specified level
func Level(minLevel slog.Level) Selector {
	return func(ctx context.Context, lvl slog.Level) tri.Tri {
		return tri.FromBool(lvl >= minLevel)
	}
}

func LevelExact(lvl slog.Level) Selector {
	return func(ctx context.Context, l slog.Level) tri.Tri {
		return tri.FromBool(l == lvl)
	}
}

type Statement func(ctx context.Context, lvl slog.Level) tri.Tri

// Selector to Statement Conversions
func undefinedLogSelector(ctx context.Context, lvl slog.Level) tri.Tri {
	return tri.Undefined
}

func (f Selector) Mute() Statement {
	// T => F
	// F => U
	// U => U
	return Statement(f.Or(undefinedLogSelector).Not())
}

func (f Selector) Show() Statement {
	// T => T
	// F => U
	// U => U
	return Statement(f.Or(undefinedLogSelector))
}

// Filter Building and Composition
// buildFilter constructs the core predicate used by DefaultShow / DefaultMute.
//
//	defaultShow – baseline decision when every selector returns tri.Undefined
//	filters     – ordered list of LogFilters (last match wins)
func buildFilter(defaultShow bool, filters ...Statement) Filter {
	return func(ctx context.Context, lvl slog.Level) bool {
		// Walk the slice in reverse so the *last* filter defined by the caller
		// is evaluated first.  As soon as we get a defined decision we can exit.
		for i := len(filters) - 1; i >= 0; i-- {
			if d := filters[i](ctx, lvl); d != tri.Undefined {
				return d.Bool(defaultShow) // early return — nothing earlier can override
			}
		}
		// No filter made a decision; fall back to the caller's default.
		return defaultShow
	}
}

// DefaultShow creates a new handlerFilter that shows logs that match the given filters.
// Later filters override earlier ones.
func DefaultShow(filters ...Statement) Filter {
	return buildFilter(true, filters...)
}

// DefaultMute creates a new handlerFilter that hides logs that match the given filters.
// Later filters override earlier ones.
func DefaultMute(filters ...Statement) Filter {
	return buildFilter(false, filters...)
}

// ============================================================================
// Filter Handler Types
// ============================================================================

type Filter func(ctx context.Context, lvl slog.Level) bool

// FilterHandler interface that can be used with the log filtering system
type FilterHandler interface {
	logmods.Handler
	// config parameter can be created with logfilter.DefaultShow(<filters...>) or logfilter.DefaultMute(<filters...>)
	Set(filter Filter)
}

// filterHandler implements the Handler interface
type filterHandler struct {
	slog.Handler
	filter atomic.Pointer[Filter]
	attrs  map[string]slog.Value
	group  string
}

var _ FilterHandler = &filterHandler{}

func (f *filterHandler) Set(filter Filter) {
	f.filter.Store(&filter)
}

func (f *filterHandler) Unwrap() slog.Handler {
	return f.Handler
}

type ValueValuer slog.Value

func (v ValueValuer) LogValue() slog.Value { return slog.Value(v) }

func (f *filterHandler) Enabled(ctx context.Context, lvl slog.Level) bool {
	// First check if the log is enabled by the inner handler, because these are less expensive to evaluate
	if !f.Handler.Enabled(ctx, lvl) {
		return false
	}
	filter := f.filter.Load()
	if filter == nil {
		return true
	}
	// Dynamically load the context with the attrs
	for k, v := range f.attrs {
		ctx = AddLogAttrToContext(ctx, k, ValueValuer(v))
	}
	result := (*filter)(ctx, lvl)
	return result
}

func (h *filterHandler) WithAttrs(as []slog.Attr) slog.Handler {
	attrs := make(map[string]slog.Value)
	// copy h.attrs to attrs
	for k, v := range h.attrs {
		attrs[k] = v
	}
	for _, a := range as {
		var key string
		if h.group != "" {
			key = h.group + "." + a.Key
		} else {
			key = a.Key
		}
		attrs[key] = a.Value
	}
	handler := filterHandler{Handler: h.Handler.WithAttrs(as), filter: atomic.Pointer[Filter]{}, attrs: attrs, group: h.group}
	handler.filter.Store(h.filter.Load())
	return &handler
}

func (h *filterHandler) WithGroup(name string) slog.Handler {
	attrs := make(map[string]slog.Value)
	// copy h.attrs to attrs
	for k, v := range h.attrs {
		attrs[k] = v
	}
	var groupName string
	if h.group != "" {
		groupName = h.group + "." + name
	} else {
		groupName = name
	}
	handler := filterHandler{Handler: h.Handler.WithGroup(name), filter: atomic.Pointer[Filter]{}, attrs: attrs, group: groupName}
	handler.filter.Store(h.filter.Load())
	return &handler
}

// Handler Wrapper
// WrapFilterHandler wraps a slog.Handler with filtering capabilities
var _ logmods.HandlerMod = WrapFilterHandler

func WrapFilterHandler(h slog.Handler) slog.Handler {
	handler := &filterHandler{
		Handler: h,
		filter:  atomic.Pointer[Filter]{},
	}
	return handler
}

// ============================================================================
// Selector Functions
// ============================================================================

// Select creates a selector that matches a log attribute with any primitive type.
// This function handles both context-based attributes (direct values) and slog WithAttrs-based
// attributes (wrapped in valueValuer). When selecting context-based attributes,
// the selector attribute name is being used to do the selection comparison.
// Values can be selected by their string representation, their slog.LogValuer
// type, or their slog.Value type that was entered into the logger with
// `logger.With(key, value)`, or into the context with `logfilter.AddLogAttrToContext(ctx, key, value)`.
func Select(key string, value any) Selector {
	var logValue slog.Value
	if v, ok := value.(slog.Value); ok {
		logValue = v
	} else if v, ok := value.(slog.LogValuer); ok {
		logValue = v.LogValue().Resolve()
	} else {
		logValue = slog.AnyValue(value)
	}
	return func(ctx context.Context, lvl slog.Level) tri.Tri {
		if v, ok := ValueFromContext[slog.LogValuer](ctx, key); ok {
			// First try direct comparison (works for context-based attributes)
			value := v.LogValue().Resolve()
			// Loosely compare values, so that the framework is more usable.
			if value.Equal(logValue) || value.String() == logValue.String() {
				return tri.True
			} else {
				return tri.False
			}
		}
		return tri.Undefined
	}
}
