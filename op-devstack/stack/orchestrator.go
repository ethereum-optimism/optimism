package stack

import (
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
)

// Lifecycle represents a controllable component by ControlPlane
type Lifecycle interface {
	Start()
	Stop()
}

type ControlAction int

const (
	Start ControlAction = iota
	Stop
)

// ControlPlane is the interface for the orchestrators to control components of the system.
type ControlPlane interface {
	SupervisorState(id SupervisorID, action ControlAction)
	L2CLNodeState(id L2CLNodeID, action ControlAction)
	FakePoSState(id L1CLNodeID, action ControlAction)
}

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

	ControlPlane() ControlPlane
}

// GateWithRemediation is an example of a test-gate that checks a system and may use an orchestrator to remediate any shortcomings.
// func GateWithRemediation(sys System, orchestrator Orchestrator) {
// step 1: check if system already does the right thing
// step 2: if not, check if orchestrator can help us
// step 3: maybe try different things, if none work, test-skip
// }

type SystemHook interface {
	// PostHydrate runs after a system is hydrated, to run any checks.
	// This may register validation that runs at the end of the test, using the sys.T().Cleanup function.
	PostHydrate(sys System)
}

// ApplyOptionLifecycle applies all option lifecycle stages to the given orchestrator
func ApplyOptionLifecycle[O Orchestrator](opt Option[O], orch O) {
	opt.BeforeDeploy(orch)
	opt.Deploy(orch)
	opt.AfterDeploy(orch)
	opt.Finally(orch)
}

// Option is used to define a change that inspects and/or changes a System during the lifecycle.
type Option[O Orchestrator] interface {
	// BeforeDeploy runs before any chain is created/deployed
	BeforeDeploy(orch O)
	// Deploy runs the deployment
	Deploy(orch O)
	// AfterDeploy runs after chains are created/deployed
	AfterDeploy(orch O)
	// Finally runs at the very end of orchestrator setup,
	// but before any test-scope is created.
	Finally(orch O)
	// SystemHook is embedded: Options may expose system hooks, to run in test-scope.
	SystemHook
}

type CommonOption = Option[Orchestrator]

// CombinedOption is a list of options.
// For each option lifecycle stage, all options are applied, left to right.
type CombinedOption[O Orchestrator] []Option[O]

var _ CommonOption = (CombinedOption[Orchestrator])(nil)

// Combine is a method to define a CombinedOption, more readable than a slice definition.
func Combine[O Orchestrator](opts ...Option[O]) CombinedOption[O] {
	return CombinedOption[O](opts)
}

// Add changes the option into a new Option that that first applies the receiver, and then the other options.
// This is a convenience for bundling options together.
func (c *CombinedOption[O]) Add(other ...Option[O]) {
	*c = append(*c, other...)
}

func (c CombinedOption[O]) BeforeDeploy(orch O) {
	for _, opt := range c {
		opt.BeforeDeploy(orch)
	}
}

func (c CombinedOption[O]) Deploy(orch O) {
	for _, opt := range c {
		opt.Deploy(orch)
	}
}

func (c CombinedOption[O]) AfterDeploy(orch O) {
	for _, opt := range c {
		opt.AfterDeploy(orch)
	}
}

func (c CombinedOption[O]) Finally(orch O) {
	for _, opt := range c {
		opt.Finally(orch)
	}
}

func (c CombinedOption[O]) PostHydrate(sys System) {
	for _, opt := range c {
		opt.PostHydrate(sys)
	}
}

// FnOption defines an option with more flexible function instances per option lifecycle stage.
// Each nil attribute is simply a no-op when not set.
type FnOption[O Orchestrator] struct {
	BeforeDeployFn func(orch O)
	DeployFn       func(orch O)
	AfterDeployFn  func(orch O)
	FinallyFn      func(orch O)
	PostHydrateFn  func(sys System)
}

var _ CommonOption = (*FnOption[Orchestrator])(nil)

func (f FnOption[O]) BeforeDeploy(orch O) {
	if f.BeforeDeployFn != nil {
		f.BeforeDeployFn(orch)
	}
}

func (f FnOption[O]) Deploy(orch O) {
	if f.DeployFn != nil {
		f.DeployFn(orch)
	}
}

func (f FnOption[O]) AfterDeploy(orch O) {
	if f.AfterDeployFn != nil {
		f.AfterDeployFn(orch)
	}
}

func (f FnOption[O]) Finally(orch O) {
	if f.FinallyFn != nil {
		f.FinallyFn(orch)
	}
}

func (f FnOption[O]) PostHydrate(sys System) {
	if f.PostHydrateFn != nil {
		f.PostHydrateFn(sys)
	}
}

// BeforeDeploy registers a function to run before the deployment stage of the orchestrator.
// This may be used to customize deployment settings.
func BeforeDeploy[O Orchestrator](fn func(orch O)) Option[O] {
	return FnOption[O]{BeforeDeployFn: fn}
}

// Deploy registers a function to run during the deployment stage of the orchestrator.
// This may be used to perform deployments.
func Deploy[O Orchestrator](fn func(orch O)) Option[O] {
	return FnOption[O]{DeployFn: fn}
}

// AfterDeploy registers a function to run after the deployment stage of the orchestrator.
// This may be used to customize the orchestrator, after having deployment configuration in place.
func AfterDeploy[O Orchestrator](fn func(orch O)) Option[O] {
	return FnOption[O]{AfterDeployFn: fn}
}

// Finally registers a function to run at the end of orchestrator setup.
// This may be used for any orchestrator post-validation,
// or to export any of the now ready orchestrator resources.
func Finally[O Orchestrator](fn func(orch O)) Option[O] {
	return FnOption[O]{FinallyFn: fn}
}

// PostHydrate hooks up an option callback to run when a new System has been hydrated by the Orchestrator.
// This is essentially a test-case preamble,
// to globally configure checks or gates that should run on the test-scope level.
// Test post-checks can be configured with sys.T().Cleanup(...).
func PostHydrate[O Orchestrator](fn func(sys System)) Option[O] {
	return FnOption[O]{PostHydrateFn: fn}
}

// MakeCommon makes the type-specific option a common option.
// If the result runs with a different orchestrator type than expected
// the actual typed option will not run.
// This can be used to mix in customizations.
// Later common options should verify the orchestrator has the properties it needs to have.
func MakeCommon[O Orchestrator](opt Option[O]) CommonOption {
	return FnOption[Orchestrator]{
		BeforeDeployFn: func(orch Orchestrator) {
			if o, ok := orch.(O); ok {
				opt.BeforeDeploy(o)
			} else {
				orch.P().Logger().Debug("BeforeDeploy option does not apply to this orchestrator type")
			}
		},
		DeployFn: func(orch Orchestrator) {
			if o, ok := orch.(O); ok {
				opt.Deploy(o)
			} else {
				orch.P().Logger().Debug("Deploy option does not apply to this orchestrator type")
			}
		},
		AfterDeployFn: func(orch Orchestrator) {
			if o, ok := orch.(O); ok {
				opt.AfterDeploy(o)
			} else {
				orch.P().Logger().Debug("AfterDeploy option does not apply to this orchestrator type")
			}
		},
		FinallyFn: func(orch Orchestrator) {
			if o, ok := orch.(O); ok {
				opt.Finally(o)
			} else {
				orch.P().Logger().Debug("Finally option does not apply to this orchestrator type")
			}
		},
		PostHydrateFn: func(sys System) {
			opt.PostHydrate(sys)
		},
	}
}
