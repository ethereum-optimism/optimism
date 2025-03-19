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

// Setup provides inputs for Option, to use during system construction
type Setup struct {
	Ctx          context.Context
	Log          log.Logger
	T            T
	Require      *require.Assertions
	System       ExtensibleSystem
	Orchestrator Orchestrator
}

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
