package sysext

import "github.com/ethereum-optimism/optimism/devnet-sdk/devstack/stack"

type ControlPlane struct {
}

func (c *ControlPlane) SupervisorState(id stack.SupervisorID, mode stack.ControlAction) {
	panic("not implemented: plug in kurtosis wrapper")
}

func (c *ControlPlane) L2CLNodeState(id stack.L2CLNodeID, mode stack.ControlAction) {
	panic("not implemented: plug in kurtosis wrapper")
}

var _ stack.ControlPlane = (*ControlPlane)(nil)
