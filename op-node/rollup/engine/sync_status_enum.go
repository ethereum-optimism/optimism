package engine

type syncStatusEnum int

const (
	syncStatusCL syncStatusEnum = iota
	// We transition between the 4 EL states linearly. We spend the majority of the time in the second & fourth.
	// We only want to EL sync if there is no finalized block & once we finish EL sync we need to mark the last block
	// as finalized so we can switch to consolidation
	// TODO(protocol-quest#91): We can restart EL sync & still consolidate if there finalized blocks on the execution client if the
	// execution client is running in archive mode. In some cases we may want to switch back from CL to EL sync, but that is complicated.
	syncStatusWillStartEL               // First if we are directed to EL sync, check that nothing has been finalized yet
	syncStatusStartedEL                 // Perform our EL sync
	syncStatusFinishedELButNotFinalized // EL sync is done, but we need to mark the final sync block as finalized
	syncStatusFinishedEL                // EL sync is done & we should be performing consolidation
)
