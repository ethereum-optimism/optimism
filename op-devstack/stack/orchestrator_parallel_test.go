package stack

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-devstack/compat"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
)

// mockOrch satisfies Orchestrator for testing parallel execution.
// All methods are stubs since tests pass nil and don't invoke them.
type mockOrch struct{}

func (m *mockOrch) P() devtest.P               { return nil }
func (m *mockOrch) Hydrate(_ ExtensibleSystem) {}
func (m *mockOrch) ControlPlane() ControlPlane { return nil }
func (m *mockOrch) Type() compat.Type          { return "test" }
func (m *mockOrch) PreHydrate(_ System)        {}
func (m *mockOrch) PostHydrate(_ System)       {}

var _ Orchestrator = (*mockOrch)(nil)

// simOption creates an option that simulates work taking `d` duration in AfterDeploy
func simOption(d time.Duration, counter *atomic.Int32) Option[*mockOrch] {
	return FnOption[*mockOrch]{
		AfterDeployFn: func(_ *mockOrch) {
			time.Sleep(d)
			counter.Add(1)
		},
	}
}

func TestInParallelRunsConcurrently(t *testing.T) {
	var counter atomic.Int32

	sleepDur := 100 * time.Millisecond
	numOpts := 4

	opts := make([]Option[*mockOrch], numOpts)
	for i := range opts {
		opts[i] = simOption(sleepDur, &counter)
	}

	// Sequential: should take ~numOpts * sleepDur
	counter.Store(0)
	seqStart := time.Now()
	combined := Combine(opts...)
	combined.AfterDeploy(nil)
	seqDur := time.Since(seqStart)
	require.Equal(t, int32(numOpts), counter.Load())

	// Parallel: should take ~sleepDur (all run concurrently)
	counter.Store(0)
	parStart := time.Now()
	par := InParallel(opts...)
	par.AfterDeploy(nil)
	parDur := time.Since(parStart)
	require.Equal(t, int32(numOpts), counter.Load())

	t.Logf("Sequential: %v", seqDur)
	t.Logf("Parallel:   %v", parDur)
	t.Logf("Speedup:    %.1fx", float64(seqDur)/float64(parDur))

	// Parallel should be at least 2x faster than sequential
	require.Less(t, parDur, seqDur/2, "parallel should be significantly faster than sequential")
}

func TestInParallelPanicPropagation(t *testing.T) {
	var counter atomic.Int32

	opts := []Option[*mockOrch]{
		simOption(10*time.Millisecond, &counter),
		FnOption[*mockOrch]{
			AfterDeployFn: func(_ *mockOrch) {
				panic("test panic")
			},
		},
		simOption(10*time.Millisecond, &counter),
	}

	par := InParallel(opts...)
	require.Panics(t, func() {
		par.AfterDeploy(nil)
	})
}

func TestInParallelSingleOption(t *testing.T) {
	var counter atomic.Int32
	opt := simOption(10*time.Millisecond, &counter)

	par := InParallel(opt)
	par.AfterDeploy(nil)
	require.Equal(t, int32(1), counter.Load())
}

func TestInParallelEmpty(t *testing.T) {
	par := InParallel[*mockOrch]()
	// Should not panic
	par.AfterDeploy(nil)
}
