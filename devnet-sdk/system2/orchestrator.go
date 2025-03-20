package system2

import (
	"context"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/log"
)

// Orchestrator is the base interface for all system orchestrators.
// It imposes some common things across all orchestrators, but may also haveoptional extensions, that not every type of backend might support.
type Orchestrator interface {

	// Example: a gate that wants funds may use the fund-account to send funds to a new account,
	// if the system doesn't already have a prefunded test account
	// FundAccount() *ecdsa.PrivateKey

}

// GateWithRemediation is an example of a test-gate that checks a system and may use an orchestrator to remediate any shortcomings.
// func GateWithRemediation(sys System, orchestrator Orchestrator) {
// step 1: check if system already does the right thing
// step 2: if not, check if orchestrator can help us
// step 3: maybe try different things, if none work, test-skip
// }

// Setup provides inputs for Option, to use during system construction.
// This object is not meant to persist longer than the execution of applicable Option(s).
// New struct-fields may be added in the future, but existing fields should not be removed for compatibility with Option implementations.
type Setup struct {
	// Ctx is the context for option execution.
	// The caller of Option should cancel the context after completion of the option.
	// The context may be interrupted if the test is aborted during option execution.
	Ctx context.Context
	// Log is a setup-wide logger, and may be used after setup by the components created by the option.
	Log log.Logger
	// T is a minimal test interface for panic-checks / assertions,
	// and may be passed down into components created by the option.
	T T
	// Require is a helper around the above T, ready to assert against.
	Require *require.Assertions
	// System is the frontend presentation of the system under test,
	// this is where component interfaces are registered to make them available to the test.
	System ExtensibleSystem
	// Orchestrator is the backend responsible for managing components,
	// and providing backend-specific options where needed, e.g. spawning new services.
	Orchestrator Orchestrator
}

// CommonConfig is a convenience method to build the config common between all components.
// Note that component constructors will decorate the logger with metadata for internal use,
// the caller of the component constructor can generally leave the logger as-is.
func (setup *Setup) CommonConfig() CommonConfig {
	return CommonConfig{
		Log: setup.Log,
		T:   setup.T,
	}
}

// Option is used to define a function that inspects and/or changes a System.
type Option func(setup *Setup)

// Append constructs a new Option that that first applies the receiver, and then the remaining options.
// This is a convenience for bundling options together.
func (fn Option) Append(other ...Option) Option {
	return func(setup *Setup) {
		fn(setup)
		for _, oFn := range other {
			oFn(setup)
		}
	}
}
