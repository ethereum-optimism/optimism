package sysgo

import "github.com/ethereum-optimism/optimism/op-devstack/stack"

// GetL1Network returns an L1 network by ID.
func (o *Orchestrator) GetL1Network(id stack.ComponentID) (*L1Network, bool) {
	return stack.RegistryGet[*L1Network](o.registry, id)
}

// GetL2Network returns an L2 network by ID.
func (o *Orchestrator) GetL2Network(id stack.ComponentID) (*L2Network, bool) {
	return stack.RegistryGet[*L2Network](o.registry, id)
}

// GetCluster returns a cluster by ID.
func (o *Orchestrator) GetCluster(id stack.ComponentID) (*Cluster, bool) {
	return stack.RegistryGet[*Cluster](o.registry, id)
}

// GetL1EL returns an L1 execution node by ID.
func (o *Orchestrator) GetL1EL(id stack.ComponentID) (L1ELNode, bool) {
	return stack.RegistryGet[L1ELNode](o.registry, id)
}

// GetL1CL returns an L1 consensus node by ID.
func (o *Orchestrator) GetL1CL(id stack.ComponentID) (*L1CLNode, bool) {
	return stack.RegistryGet[*L1CLNode](o.registry, id)
}

// GetL2CL returns an L2 consensus node by ID.
func (o *Orchestrator) GetL2CL(id stack.ComponentID) (L2CLNode, bool) {
	return stack.RegistryGet[L2CLNode](o.registry, id)
}

// GetSupervisor returns a supervisor by ID.
func (o *Orchestrator) GetSupervisor(id stack.ComponentID) (Supervisor, bool) {
	return stack.RegistryGet[Supervisor](o.registry, id)
}

// GetOPRBuilder returns an OPR builder node by ID.
func (o *Orchestrator) GetOPRBuilder(id stack.ComponentID) (*OPRBuilderNode, bool) {
	return stack.RegistryGet[*OPRBuilderNode](o.registry, id)
}

// GetRollupBoost returns a rollup-boost node by ID.
func (o *Orchestrator) GetRollupBoost(id stack.ComponentID) (*RollupBoostNode, bool) {
	return stack.RegistryGet[*RollupBoostNode](o.registry, id)
}
