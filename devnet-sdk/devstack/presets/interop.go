package presets

import (
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/dsl"
	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/stack"
	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/sysgo"
)

type SimpleInterop struct {
	Log        log.Logger
	System     *dsl.System
	Supervisor stack.Supervisor
	// Nodes / handles from DSL package go here
}

func NewSimpleInterop(t stack.T, opts ...stack.Option) *SimpleInterop {
	setup := NewSetup(t,
		WithTestLogger(),
		WithEmptySystem(),
		WithGlobalOrchestrator())

	for _, opt := range opts {
		opt(setup)
	}

	ids, opt := sysgo.DefaultInteropSystem(sysgo.ContractPaths{
		FoundryArtifacts: "../../../packages/contracts-bedrock/forge-artifacts",
		SourceMap:        "../../../packages/contracts-bedrock",
	})
	opt(setup)

	super := setup.System.Supervisor(ids.Supervisor)

	return &SimpleInterop{
		Log:        setup.Log,
		System:     dsl.Hydrate(setup.System),
		Supervisor: super,
	}
}
