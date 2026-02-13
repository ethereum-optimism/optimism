package stack

import (
	"github.com/ethereum-optimism/optimism/op-service/apis"
)

// L2ELNode is a L2 ethereum execution-layer node
type L2ELNode interface {
	L2EthClient() apis.L2EthClient
	L2EngineClient() apis.EngineClient

	ELNode
}
