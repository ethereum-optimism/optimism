package stack

import (
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// L2CLNodeID identifies a L2CLNode by name and chainID, is type-safe, and can be value-copied and used as map key.
type L2CLNodeID ComponentID

const L2CLNodeKind Kind = "L2CLNode"

func NewL2CLNodeID(key string) ComponentID {
	return ComponentID{
		Kind: L2CLNodeKind,
		Key:  key,
	}
}

func SortL2CLNodes(elems []L2CLNode) []L2CLNode {
	return copyAndSort(elems, func(a, b L2CLNode) bool {
		return isLess(a.ID(), b.ID())
	})
}

// L2CLNode is a L2 ethereum consensus-layer node
type L2CLNode interface {
	Common
	ID() ComponentID
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

type LinkableL2CLNode interface {
	// Links the nodes. Does not make any backend changes, just registers the EL as connected to this CL.
	LinkEL(el L2ELNode)
	LinkRollupBoostNode(rollupBoostNode RollupBoostNode)
	LinkOPRBuilderNode(oprb OPRBuilderNode)
}
