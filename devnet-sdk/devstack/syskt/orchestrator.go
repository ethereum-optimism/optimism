package syskt

import (
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/devnet-sdk/descriptors"
	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/stack"
)

type OrchestratorOption func(*Orchestrator)

type Orchestrator struct {
	t   stack.T
	log log.Logger

	env *descriptors.DevnetEnvironment

	usePrivatePorts    bool
	useEagerRPCClients bool
}

var _ stack.Orchestrator = (*Orchestrator)(nil)

func NewOrchestrator(t stack.T, log log.Logger) *Orchestrator {
	return &Orchestrator{t: t, log: log}
}

func (o *Orchestrator) T() stack.T {
	return o.t
}

func (o *Orchestrator) Log() log.Logger {
	return o.log
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
