package presets

import (
	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/dsl"
	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/stack"
	"github.com/ethereum/go-ethereum/log"
)

type SimpleInterop struct {
	Log    log.Logger
	System *dsl.System
	// Nodes / handles from DSL package go here
}

func NewSimpleInterop(t stack.T) *SimpleInterop {
	setup := NewSetup(t,
		WithTestLogger(),
		WithEmptySystem(),
		WithGlobalOrchestrator())

	// TODO apply options

	setup.Log.Info("hello world")

	return &SimpleInterop{
		Log:    setup.Log,
		System: dsl.Hydrate(setup.System),
	}
}
