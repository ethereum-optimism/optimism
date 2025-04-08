package sysext

import "github.com/ethereum-optimism/optimism/devnet-sdk/devstack/stack"

type ControlPanel struct {
	o *Orchestrator
}

func (c *ControlPanel) SupervisorState(id stack.SupervisorID, mode stack.Mode) {
	// TODO kurtosis command
}

func (c *ControlPanel) L2CLNodeState(id stack.L2CLNodeID, mode stack.Mode) {
	// TODO kurtosis command
}

var _ stack.ControlPanel = (*ControlPanel)(nil)
