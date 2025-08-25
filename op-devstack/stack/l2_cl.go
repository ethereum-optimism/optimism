package stack

import (
	"log/slog"

	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// L2CLNodeID identifies a L2CLNode by name and chainID, is type-safe, and can be value-copied and used as map key.
type L2CLNodeID idWithChain

var _ IDWithChain = L2CLNodeID{}

const L2CLNodeKind Kind = "L2CLNode"

func NewL2CLNodeID(key string, chainID eth.ChainID) L2CLNodeID {
	return L2CLNodeID{
		key:     key,
		chainID: chainID,
	}
}

func (id L2CLNodeID) String() string {
	return idWithChain(id).string(L2CLNodeKind)
}

func (id L2CLNodeID) ChainID() eth.ChainID {
	return id.chainID
}

func (id L2CLNodeID) Kind() Kind {
	return L2CLNodeKind
}

func (id L2CLNodeID) Key() string {
	return id.key
}

func (id L2CLNodeID) LogValue() slog.Value {
	return slog.StringValue(id.String())
}

func (id L2CLNodeID) MarshalText() ([]byte, error) {
	return idWithChain(id).marshalText(L2CLNodeKind)
}

func (id *L2CLNodeID) UnmarshalText(data []byte) error {
	return (*idWithChain)(id).unmarshalText(L2CLNodeKind, data)
}

func SortL2CLNodeIDs(ids []L2CLNodeID) []L2CLNodeID {
	return copyAndSort(ids, func(a, b L2CLNodeID) bool {
		return lessIDWithChain(idWithChain(a), idWithChain(b))
	})
}

func SortL2CLNodes(elems []L2CLNode) []L2CLNode {
	return copyAndSort(elems, func(a, b L2CLNode) bool {
		return lessIDWithChain(idWithChain(a.ID()), idWithChain(b.ID()))
	})
}

var _ L2CLMatcher = L2CLNodeID{}

func (id L2CLNodeID) Match(elems []L2CLNode) []L2CLNode {
	return findByID(id, elems)
}

// L2CLNode is a L2 ethereum consensus-layer node
type L2CLNode interface {
	Common
	ID() L2CLNodeID

	ClientRPC() client.RPC
	RollupAPI() apis.RollupClient
	P2PAPI() apis.P2PClient
	InteropRPC() (endpoint string, jwtSecret eth.Bytes32)

	// ELs returns the engine(s) that this L2CLNode is connected to.
	// This may be empty, if the L2CL is not connected to any.
	ELs() []L2ELNode
}

type LinkableL2CLNode interface {
	// Links the nodes. Does not make any backend changes, just registers the EL as connected to this CL.
	LinkEL(el L2ELNode)
}
