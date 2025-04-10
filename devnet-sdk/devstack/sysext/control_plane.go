package sysext

import "github.com/ethereum-optimism/optimism/devnet-sdk/devstack/stack"

type ControlPlane struct {
	o *Orchestrator
}

func (c *ControlPlane) SupervisorState(id stack.SupervisorID, mode stack.Mode) {
	// TODO kurtosis command
}

func (c *ControlPlane) L2CLNodeState(id stack.L2CLNodeID, mode stack.Mode) {
	// TODO kurtosis command
}

var _ stack.ControlPlane = (*ControlPlane)(nil)
