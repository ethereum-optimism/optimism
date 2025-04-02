package presets

import (
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/dsl"
	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/stack"
	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/sysgo"
)

type SimpleInterop struct {
	Log        log.Logger
	Supervisor *dsl.Supervisor
}

func NewSimpleInterop(t stack.T, opts ...stack.Option) *SimpleInterop {
	setup := NewSetup(t,
		WithTestLogger(),
		WithEmptySystem(),
		WithGlobalOrchestrator())

	for _, opt := range opts {
		opt(setup)
	}

	contracts, err := contractPaths()
	setup.Require.NoError(err, "could not get contract paths")
	ids, opt := sysgo.DefaultInteropSystem(contracts)
	opt(setup)

	sys := dsl.Hydrate(setup)
	return &SimpleInterop{
		Log:        setup.Log,
		Supervisor: sys.Supervisor(ids.Supervisor),
	}
}
