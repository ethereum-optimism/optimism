package sysext

import (
	"os"

	"github.com/ethereum-optimism/optimism/devnet-sdk/descriptors"
	"github.com/ethereum-optimism/optimism/devnet-sdk/shell/env"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
)

const defaultDevnetUrl = "kt://interop-devnet"

type OrchestratorOption func(*Orchestrator)

type Orchestrator struct {
	p devtest.P

	env *env.DevnetEnv

	usePrivatePorts    bool
	useEagerRPCClients bool

	controlPlane *ControlPlane
	useDirectCnx bool

	// sysHook is called after hydration of a new test-scope system frontend,
	// essentially a test-case preamble.
	sysHook stack.SystemHook
}

var _ stack.Orchestrator = (*Orchestrator)(nil)

func (o *Orchestrator) ControlPlane() stack.ControlPlane {
	return o.controlPlane
}

func NewOrchestrator(p devtest.P, sysHook stack.SystemHook) *Orchestrator {
	url := os.Getenv(env.EnvURLVar)
	if url == "" {
		p.Logger().Warn("No devnet URL specified, using default", "default", defaultDevnetUrl)
		url = defaultDevnetUrl
	}
	env, err := env.LoadDevnetFromURL(url)
	p.Require().NoError(err, "Error loading devnet environment")
	p.Require().NotNil(env, "Error loading devnet environment: DevnetEnv is nil")
	p.Require().NotNil(env.Env, "Error loading devnet environment: DevnetEnvironment is nil")

	orch := &Orchestrator{
		env:     env,
		p:       p,
		sysHook: sysHook,
	}
	orch.controlPlane = &ControlPlane{
		o: orch,
	}
	return orch
}

func (o *Orchestrator) P() devtest.P {
	return o.p
}

func (o *Orchestrator) Hydrate(sys stack.ExtensibleSystem) {
	if o.env == nil || o.env.Env == nil {
		panic("orchestrator not properly initialized: env is nil")
	}

	o.sysHook.PreHydrate(sys)
	o.hydrateL1(sys)
	o.hydrateSuperchain(sys)
	o.hydrateClustersMaybe(sys)
	o.hydrateSupervisorsMaybe(sys)
	o.hydrateTestSequencersMaybe(sys)
	for _, l2Net := range o.env.Env.L2 {
		o.hydrateL2(l2Net, sys)
	}
	o.sysHook.PostHydrate(sys)
}

func isInterop(env *descriptors.DevnetEnvironment) bool {
	for _, feature := range env.Features {
		if feature == FeatureInterop {
			return true
		}
	}
	return false
}

func (o *Orchestrator) isInterop() bool {
	// Ugly hack to ensure we can use L2[0] for supervisor
	// Ultimately this should be removed.
	return isInterop(o.env.Env) && len(o.env.Env.L2) > 0
}

func WithPrivatePorts() OrchestratorOption {
	return func(orchestrator *Orchestrator) {
		orchestrator.usePrivatePorts = true
	}
}

func WithEagerRPCClients() OrchestratorOption {
	return func(orchestrator *Orchestrator) {
		orchestrator.useEagerRPCClients = true
	}
}

func WithDirectConnections() OrchestratorOption {
	return func(orchestrator *Orchestrator) {
		orchestrator.useDirectCnx = true
	}
}
