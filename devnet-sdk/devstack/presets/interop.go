package presets

import (
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/devtest"
	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/dsl"
	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/shim"
	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/stack"
	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/sysgo"
)

type SimpleInterop struct {
	Log        log.Logger
	T          devtest.T
	Supervisor *dsl.Supervisor
}

func NewSimpleInterop(dest *TestSetup[*SimpleInterop]) stack.Option {
	return func(orch stack.Orchestrator) {
		_, ok := orch.(*sysgo.Orchestrator)
		orch.P().Require().True(ok, "only sysgo supported for now")

		contracts, err := contractPaths()
		orch.P().Require().NoError(err, "could not get contract paths")
		var ids sysgo.DefaultInteropSystemIDs
		opt := sysgo.DefaultInteropSystem(contracts, &ids)
		opt(orch)
		*dest = func(t devtest.T) *SimpleInterop {
			system := shim.NewSystem(t)
			orch.Hydrate(system)

			sys := dsl.Hydrate(t, system)
			return &SimpleInterop{
				Log:        t.Logger(),
				T:          t,
				Supervisor: sys.Supervisor(ids.Supervisor),
			}
		}
	}
}
