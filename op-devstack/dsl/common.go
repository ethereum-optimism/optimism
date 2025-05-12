package dsl

import (
	"context"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
)

// commonImpl provides a set of common values and methods inherited by all DSL structs.
// These should be kept very minimal.
// No public methods or fields should be exposed.
// This aims to make interfacing with the common devtest functionality less verbose.
// The test internal values should never change.
// Instead, a new component DSL binding may be initialized, for usage in a new (sub-)test-scope.
type commonImpl struct {
	// Ctx is the context for test execution.
	ctx context.Context
	// log is the component-specific logger instance.
	log log.Logger
	// T is a minimal test interface for panic-checks / assertions.
	t devtest.T
	// Require is a helper around the above T, ready to assert against.
	require *require.Assertions
}

func commonFromT(t devtest.T) commonImpl {
	return commonImpl{
		ctx:     t.Ctx(),
		log:     t.Logger(),
		t:       t,
		require: t.Require(),
	}
}
