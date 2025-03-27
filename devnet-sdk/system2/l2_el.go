package system2

import "github.com/stretchr/testify/require"

// L2ELNodeID identifies a L2ELNode by name and chainID, is type-safe, and can be value-copied and used as map key.
type L2ELNodeID idWithChain

const L2ELNodeKind Kind = "L2ELNode"

func (id L2ELNodeID) String() string {
	return idWithChain(id).string(L2ELNodeKind)
}

func (id L2ELNodeID) MarshalText() ([]byte, error) {
	return idWithChain(id).marshalText(L2ELNodeKind)
}

func (id *L2ELNodeID) UnmarshalText(data []byte) error {
	return (*idWithChain)(id).unmarshalText(L2ELNodeKind, data)
}

func SortL2ELNodeIDs(ids []L2ELNodeID) []L2ELNodeID {
	return copyAndSort(ids, func(a, b L2ELNodeID) bool {
		return lessIDWithChain(idWithChain(a), idWithChain(b))
	})
}

// L2ELNode is a L2 ethereum execution-layer node
type L2ELNode interface {
	ID() L2ELNodeID

	ELNode
}

type L2ELNodeConfig struct {
	ELNodeConfig
	ID L2ELNodeID
}

type rpcL2ELNode struct {
	rpcELNode

	id L2ELNodeID
}

var _ L2ELNode = (*rpcL2ELNode)(nil)

func NewL2ELNode(cfg L2ELNodeConfig) L2ELNode {
	require.Equal(cfg.T, cfg.ID.ChainID, cfg.ELNodeConfig.ChainID, "chainID must be configured to match node chainID")
	cfg.Log = cfg.Log.New("chainID", cfg.ID.ChainID, "id", cfg.ID)
	return &rpcL2ELNode{
		rpcELNode: newRpcELNode(cfg.ELNodeConfig),
		id:        cfg.ID,
	}
}

func (r *rpcL2ELNode) ID() L2ELNodeID {
	return r.id
}
