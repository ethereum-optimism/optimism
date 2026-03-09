package stack

import (
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// L2CLNode is a L2 ethereum consensus-layer node
type L2CLNode interface {
	Common
	ChainID() eth.ChainID

	ClientRPC() client.RPC
	RollupAPI() apis.RollupClient
	P2PAPI() apis.P2PClient
	InteropRPC() (endpoint string, jwtSecret eth.Bytes32)
	UserRPC() string

	// ELs returns the engine(s) that this L2CLNode is connected to.
	// This may be empty, if the L2CL is not connected to any.
	ELs() []L2ELNode
	RollupBoostNodes() []RollupBoostNode
	OPRBuilderNodes() []OPRBuilderNode

	ELClient() apis.EthClient
}
