package stack

// L1ELNode is a L1 ethereum execution-layer node
type L1ELNode interface {
	ID() ComponentID

	ELNode
}
