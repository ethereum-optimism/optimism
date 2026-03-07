package adapters

import (
	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/graph"
	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/model"
)

// TestRunner executes tests for a specific language.
type TestRunner interface {
	// Run executes the given targets under the given configuration.
	Run(targets []model.Target, config model.Configuration, opts RunOptions) ([]model.TestResult, error)

	// RunOne re-executes a single named test. Used by retry/classification logic.
	RunOne(test model.TestIdentifier, config model.Configuration, opts RunOptions) (model.TestResult, error)

	// Language returns the adapter's language identifier.
	Language() string
}

// RunOptions configures a test run.
type RunOptions struct {
	Timeout     int               // seconds
	Parallelism int
	WorkDir     string
	ExtraEnv    map[string]string
}

// Adapter bundles a dependency graph and test runner for a language.
type Adapter struct {
	Graph  graph.DependencyGraph
	Runner TestRunner
}
