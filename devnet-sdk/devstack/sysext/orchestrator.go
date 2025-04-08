package sysext

import (
	"os"

	"github.com/ethereum-optimism/optimism/devnet-sdk/descriptors"
	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/devtest"
	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/stack"
	"github.com/ethereum-optimism/optimism/devnet-sdk/shell/env"
)

const defaultDevnetUrl = "kt://interop-devnet"

type OrchestratorOption func(*Orchestrator)

type Orchestrator struct {
	p devtest.P

	env *descriptors.DevnetEnvironment

	usePrivatePorts    bool
	useEagerRPCClients bool
}

var _ stack.Orchestrator = (*Orchestrator)(nil)

func NewOrchestrator(p devtest.P) *Orchestrator {
	url := os.Getenv(env.EnvURLVar)
	if url == "" {
		p.Logger().Warn("No devnet URL specified, using default", "default", defaultDevnetUrl)
		url = defaultDevnetUrl
	}
	env, err := env.LoadDevnetFromURL(url)
	p.Require().NoError(err, "Error loading devnet environment")
	return &Orchestrator{env: &env.Config, p: p}
}

func (o *Orchestrator) P() devtest.P {
	return o.p
}

func (o *Orchestrator) Hydrate(sys stack.ExtensibleSystem) {
	o.hydrateL1(sys)
	o.hydrateSuperchain(sys)
	o.hydrateClusterMaybe(sys)
	o.hydrateSupervisorMaybe(sys)
	for _, l2Net := range o.env.L2 {
		o.hydrateL2(l2Net, sys)
	}
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
	return isInterop(o.env) && len(o.env.L2) > 0
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
