package stack

import (
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/client"
)

// OPRBuilderNode is a L2 ethereum execution-layer node
type OPRBuilderNode interface {
	ID() ComponentID
	L2EthClient() apis.L2EthClient
	L2EngineClient() apis.EngineClient
	FlashblocksClient() *client.WSClient

	ELNode
}
