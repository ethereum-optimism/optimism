package presets

import (
	"fmt"
	"log/slog"

	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-service/logfilter"
	"github.com/ethereum-optimism/optimism/op-service/logmods"
)

// WithLogFiltersReset removes all filters, and puts a global minimum-log-level filter in place.
func WithLogFiltersReset(globalMinLevel slog.Level) stack.CommonOption {
	fn := func(h logfilter.Handler) {
		h.Set(logfilter.Minimum(globalMinLevel))
	}
	return stack.Combine(
		withPkgLogFiltering(fn),
		withTestLogFiltering(fn),
	)
}

// WithLogFilters applies the log filter to the orchestrator and each test-scope
func WithLogFilters(filters ...logfilter.LogFilter) stack.CommonOption {
	fn := func(h logfilter.Handler) {
		h.Add(logfilter.Combine(filters...))
	}
	return stack.Combine(
		withPkgLogFiltering(fn),
		withTestLogFiltering(fn),
	)
}

// WithPkgLogFilters applies the log filters to test-scope interactions
// (i.e. to things like DSL interactions, not to background services).
func WithPkgLogFilters(filters ...logfilter.LogFilter) stack.CommonOption {
	fn := func(h logfilter.Handler) {
		h.Add(logfilter.Combine(filters...))
	}
	return withPkgLogFiltering(fn)
}

// WithTestLogFilters applies the log filters to test-scope interactions
// (i.e. to things like DSL interactions, not to background services).
func WithTestLogFilters(filters ...logfilter.LogFilter) stack.CommonOption {
	fn := func(h logfilter.Handler) {
		h.Add(logfilter.Combine(filters...))
	}
	return withTestLogFiltering(fn)
}

// withPkgLogFiltering creates an option to apply changes to the log-handlers of package-level logger and test-scopes.
func withPkgLogFiltering(fn func(h logfilter.Handler)) stack.CommonOption {
	return stack.BeforeDeploy(func(orch stack.Orchestrator) {
		logger := orch.P().Logger()
		h := logger.Handler()
		filterHandler, ok := logmods.FindHandler[logfilter.Handler](h)
		if !ok {
			logger.Warn("Cannot apply log-filters to pkg-scope log-handler", "type", fmt.Sprintf("%T", h))
			return
		}
		fn(filterHandler)
	})
}

func withTestLogFiltering(fn func(h logfilter.Handler)) stack.CommonOption {
	return stack.PreHydrate[stack.Orchestrator](func(sys stack.System) {
		logger := sys.T().Logger()
		h := logger.Handler()
		filterHandler, ok := logmods.FindHandler[logfilter.Handler](h)
		if !ok {
			logger.Warn("Cannot apply log-filters to test-scope log-handler", "type", fmt.Sprintf("%T", h))
			return
		}
		fn(filterHandler)
	})
}
