package systemext

import (
	"github.com/ethereum-optimism/optimism/devnet-sdk/descriptors"
	"github.com/ethereum-optimism/optimism/devnet-sdk/system2"
)

type OrchestratorOption func(*Orchestrator)

type Orchestrator struct {
	env *descriptors.DevnetEnvironment

	usePrivatePorts    bool
	useEagerRPCClients bool
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

var _ system2.Orchestrator = (*Orchestrator)(nil)
