package stack

// L1Network represents a L1 chain, a collection of configuration and node resources.
type L1Network interface {
	Network

	L1ELNodes() []L1ELNode
	L1CLNodes() []L1CLNode
}
