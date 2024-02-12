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

	// TODO a hold function to reserve a named resource.
	// Held resources should automatically be freed at the end of the test scope.
	// Hold(name)

	// Run runs a sub-test. Sub-tests may be used to structure tests, and allow filtering of sub-test cases.
	// For backend parametrization, see Select, to signal parametrization in the environment.
	Run(name string, fn func(t Testing))

	// Parameter returns what the currently configured parameter value is for the given parameter name, if any.
	Parameter(name string) (value string, ok bool)

	// Select selects a parameter from the given options, based on the test ParameterSelector.
	// Options may restrict what the test is able to run.
	//
	// The ParameterSelector may return a subset of the options:
	// the first is continued with as default, and the current test will be repeated with the others, if any.
	Select(name string, options ...string) string

	// Value requests the ParameterSelector to pick a value for the given parameter,
	// or retrieves the paramter if it is already set.
	//
	// The ParameterSelector may return more than 1 value:
	// the first is continued with as default, and the current test will be repeated with the others, if any.
	Value(name string) string
}

type ParameterSelector interface {
	// Select selects which options to run with.
	//
	// The returned slice must be non-empty, or the test will be skipped.
	// The first entry will be continued with in the default execution path.
	// The remaining entries will be completed in sub-tests, if any.
	Select(name string, options []string) []string

	// Values provides values for the given parameter,
	// without limiting to a specific prescribed set of options.
	//
	// The returned slice must be non-empty, or the test will be skipped.
	// The first entry will be continued with in the default execution path.
	// The remaining entries will be completed in sub-tests, if any.
	Values(name string) []string
}

// Select is a generic helper method, to do typed Testing.Select calls.
func Select[E fmt.Stringer](t Testing, name string, options ...E) E {
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
