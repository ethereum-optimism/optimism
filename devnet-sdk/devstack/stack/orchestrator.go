package stack

import (
	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/devtest"
)

// Orchestrator is the base interface for all system orchestrators.
// It imposes some common things across all orchestrators, but may also have optional extensions, that not every type of backend might support.
type Orchestrator interface {
	// P is the test-handle of the orchestrator.
	// This may not be a Go-test handle.
	// Orchestrators may be instantiated by dev-tools or test-package TestMain functions.
	P() devtest.P

	// Hydrate adds all services that the orchestrator is aware of to the given system.
	// An orchestrator may be asked to hydrate different systems, one for each test.
	Hydrate(sys ExtensibleSystem)
}

// GateWithRemediation is an example of a test-gate that checks a system and may use an orchestrator to remediate any shortcomings.
// func GateWithRemediation(sys System, orchestrator Orchestrator) {
// step 1: check if system already does the right thing
// step 2: if not, check if orchestrator can help us
// step 3: maybe try different things, if none work, test-skip
// }

// Option is used to define a function that inspects and/or changes a System.
type Option func(orch Orchestrator)

// Add changes the option into a new Option that that first applies the receiver, and then the other options.
// This is a convenience for bundling options together.
func (fn *Option) Add(other ...Option) {
	inner := *fn
	*fn = func(orch Orchestrator) {
		inner(orch)
		for _, oFn := range other {
			oFn(orch)
		}
	}
}
