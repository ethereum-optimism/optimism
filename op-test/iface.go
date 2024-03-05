package op_test

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils"
)

type Testing interface {
	e2eutils.TestingBase

	// Ctx returns the current testing-scope context.
	// Internally this context hosts the parameters of the tests.
	// The context may be replaced when test-parameters are chosen.
	// The context is canceled when the test-scope completes,
	// including all the parameterized sub-tests (if any).
	Ctx() context.Context

	// Logger returns a testlog logger, unique to this test-scope.
	// The same logger is returned when repeatedly called.
	Logger() log.Logger

	// Parameter returns what the currently configured parameter value is for the given parameter name, if any.
	//
	// A parameter value is scoped to the sub-test it was set in.
	Parameter(name string) (value string, ok bool)
}

type Executor interface {
	Testing

	// Run runs a sub-test, if it matches the test-filter.
	Run(name string, fn func(t Executor))

	// TODO a hold function to reserve a named resource.
	// Held resources should automatically be freed at the end of the test scope.
	// Hold(name)
}

type Planner interface {
	Testing

	// Plan runs a sub-test planner. Sub-tests may be used to structure tests, and allow filtering of sub-test cases.
	// For parametrization, see Select, to signal parametrization in the environment.
	Plan(name string, fn func(t Planner))

	// Run takes the test-plan that was created thus-far, and either executes it, or persists it for later execution.
	Run(name string, fn func(t Executor))

	// Select selects a parameter from the given options, based on the test ParameterSelector.
	// Options may restrict what the test is able to run.
	//
	// Option value "*" may be used as a catch-all, for any parameter value.
	//
	// The selected parameter value is scoped to the sub-test it was selected from.
	//
	// The ParameterSelector may return a subset of the options:
	// the first is continued with as default, and the current test will be repeated with the others, if any.
	Select(name string, options ...string) string
}

type ParameterSelector interface {
	// Select selects which options to run with.
	//
	// If "any" value is requested, then "*" can be used as wildcard.
	//
	// The returned slice must be non-empty, or the test will be skipped.
	// The first entry will be continued with in the default execution path.
	// The remaining entries will be completed in sub-tests, if any.
	Select(name string, options []string) []string
}

// Select is a generic helper method, to do typed Testing.Select calls.
func Select[E fmt.Stringer](t Planner, name string, options ...E) E {
	input := make([]string, 0, len(options))
	for _, opt := range options {
		input = append(input, opt.String())
	}
	output := t.Select(name, input...)
	for _, opt := range options {
		if opt.String() == output {
			return opt
		}
	}
	t.Fatalf("selected unknown option of type %q: %q", name, output)
	panic("unknown option")
}
