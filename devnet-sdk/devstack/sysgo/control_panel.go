package sysgo

import "github.com/ethereum-optimism/optimism/devnet-sdk/devstack/stack"

type ControlPanel struct {
	o *Orchestrator
}

func (c *ControlPanel) SupervisorState(id stack.SupervisorID, mode stack.Mode) {
	// TODO
	s, ok := c.o.supervisors.Get(id)
	c.o.P().Require().True(ok, "need supervisor to change state")
	switch mode {
	case stack.Stopped:
		s.stop()
	case stack.Started:
		s.start()
	}
}

func (c *ControlPanel) L2CLNodeState(id stack.L2CLNodeID, mode stack.Mode) {
	// above supervisor handle can be behind {start() stop()} interface
}

var _ stack.ControlPanel = (*ControlPanel)(nil)
