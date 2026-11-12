package stack

import "context"

type Lifecycle interface {
	Start()
	Stop()
}

// Wipeable is implemented by nodes whose chain data can be discarded, so that the
// next Start begins from genesis. Stop alone does not guarantee this: whether a
// node keeps its data across a restart depends on the implementation.
type Wipeable interface {
	// Wipe discards the node's chain data. The node must be stopped.
	Wipe()
}

type ControlledLifecycle interface {
	StartControlled(ctx context.Context) error
	StopControlled(ctx context.Context) error
	Running() bool
}
