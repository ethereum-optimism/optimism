package presets

import (
	"fmt"
	"log/slog"

	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-service/log/logfilter"
	"github.com/ethereum-optimism/optimism/op-service/logmods"
)

// WithLogLevel sets the global minimum log-level.
func WithLogLevel(minLevel slog.Level) stack.CommonOption {
	fn := func(h logfilter.FilterHandler) {
		h.Set(logfilter.DefaultMute(logfilter.Level(minLevel).Show()))
	}
	return stack.Combine(
		withPkgLogFiltering(fn),
		withTestLogFiltering(fn),
	)
}

// WithLogFilter replaces the default log filter with the provided additive
// filters. This completely overrides the default INFO-level filtering.
func WithLogFilter(filter logfilter.Filter) stack.CommonOption {
	fn := func(h logfilter.FilterHandler) {
		h.Set(filter)
	}
	return stack.Combine(
		withPkgLogFiltering(fn),
		withTestLogFiltering(fn),
	)
}

// WithPkgLogFilter applies the log filters to package-scope interactions
// (i.e. to things like DSL interactions, not to background services).
// Calling this overrides the default INFO-level filtering.
func WithPkgLogFilter(filter logfilter.Filter) stack.CommonOption {
	fn := func(h logfilter.FilterHandler) {
		h.Set(filter)
	}
	return withPkgLogFiltering(fn)
}

// WithTestLogFilter applies the log filters to test-scope interactions
// (i.e. to things like DSL interactions, not to background services).
// Calling this overrides the default INFO-level filtering.
func WithTestLogFilter(filter logfilter.Filter) stack.CommonOption {
	fn := func(h logfilter.FilterHandler) {
		h.Set(filter)
	}
	return withTestLogFiltering(fn)
}

// withPkgLogFiltering creates an option to apply changes to the log-handlers of
// package-level logger and test-scopes.
func withPkgLogFiltering(fn func(h logfilter.FilterHandler)) stack.CommonOption {
	return stack.BeforeDeploy(func(orch stack.Orchestrator) {
		logger := orch.P().Logger()
		h := logger.Handler()
		filterHandler, ok := logmods.FindHandler[logfilter.FilterHandler](h)
		if !ok {
			logger.Warn("Cannot apply log-filters to pkg-scope log-handler", "type", fmt.Sprintf("%T", h))
			return
		}
		fn(filterHandler)
	})
}

func withTestLogFiltering(fn func(h logfilter.FilterHandler)) stack.CommonOption {
	return stack.PreHydrate[stack.Orchestrator](func(sys stack.System) {
		logger := sys.T().Logger()
		h := logger.Handler()
		filterHandler, ok := logmods.FindHandler[logfilter.FilterHandler](h)
		if !ok {
			logger.Warn("Cannot apply log-filters to test-scope log-handler", "type", fmt.Sprintf("%T", h))
			return
		}
		fn(filterHandler)
	})
}
