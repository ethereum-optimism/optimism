package stack

import (
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/client"
)

// RollupBoostNode is a shim service between an L2 consensus-layer node and an L2 ethereum execution-layer node
type RollupBoostNode interface {
	ID() ComponentID
	L2EthClient() apis.L2EthClient
	L2EngineClient() apis.EngineClient
	FlashblocksClient() *client.WSClient

	ELNode
}
