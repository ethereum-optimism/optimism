package stack

// L1Network represents a L1 chain, a collection of configuration and node resources.
type L1Network interface {
	Network
	ID() ComponentID

	L1ELNode(m L1ELMatcher) L1ELNode
	L1CLNode(m L1CLMatcher) L1CLNode

	L1ELNodeIDs() []ComponentID
	L1CLNodeIDs() []ComponentID

	L1ELNodes() []L1ELNode
	L1CLNodes() []L1CLNode
}

type ExtensibleL1Network interface {
	ExtensibleNetwork
	L1Network
	AddL1ELNode(v L1ELNode)
	AddL1CLNode(v L1CLNode)
}
