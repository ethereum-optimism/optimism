package presets

import "github.com/ethereum-optimism/optimism/op-devstack/stack"

func WithTimeTravel() stack.Option[stack.Orchestrator] {
	return stack.Combine(
		stack.BeforeDeploy[stack.Orchestrator](func(orch stack.Orchestrator) {
			ttOrch, ok := orch.(stack.TimeTravelOrchestrator)
			if !ok {
				return
			}
			ttOrch.EnableTimeTravel()
		}),
		stack.PostHydrate[stack.Orchestrator](func(sys stack.System) {
			sys.L1Networks()
			ttSys, ok := sys.(stack.TimeTravelSystem)
			sys.T().Gate().True(ok, "Requires system supporting time travel")
			sys.T().Gate().True(ttSys.TimeTravelEnabled(), "Time travel must be enabled")
		}),
	)
}
