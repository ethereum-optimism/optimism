package stack

import (
	"github.com/ethereum/go-ethereum/params"
)

// L1Network represents a L1 chain, a collection of configuration and node resources.
type L1Network interface {
	Network

	ChainConfig() *params.ChainConfig

	L1ELNodes() []L1ELNode
	L1CLNodes() []L1CLNode
}
