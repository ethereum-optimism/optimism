package system2

import "github.com/stretchr/testify/require"

// L1ELNodeID identifies a L1ELNode by name and chainID, is type-safe, and can be value-copied and used as map key.
type L1ELNodeID idWithChain

const L1ELNodeKind Kind = "L1ELNode"

func (id L1ELNodeID) String() string {
	return idWithChain(id).string(L1ELNodeKind)
}

func (id L1ELNodeID) MarshalText() ([]byte, error) {
	return idWithChain(id).marshalText(L1ELNodeKind)
}

func (id *L1ELNodeID) UnmarshalText(data []byte) error {
	return (*idWithChain)(id).unmarshalText(L1ELNodeKind, data)
}

func SortL1ELNodeIDs(ids []L1ELNodeID) []L1ELNodeID {
	return copyAndSort(ids, func(a, b L1ELNodeID) bool {
		return lessIDWithChain(idWithChain(a), idWithChain(b))
	})
}

// L1ELNode is a L1 ethereum execution-layer node
type L1ELNode interface {
	ID() L1ELNodeID

	ELNode
}

type L1ELNodeConfig struct {
	ELNodeConfig
	ID L1ELNodeID
}

type rpcL1ELNode struct {
	rpcELNode
	id L1ELNodeID
}

var _ L1ELNode = (*rpcL1ELNode)(nil)

func NewL1ELNode(cfg L1ELNodeConfig) L1ELNode {
	require.Equal(cfg.T, cfg.ID.ChainID, cfg.ELNodeConfig.ChainID, "chainID must be configured to match node chainID")
	cfg.Log = cfg.Log.New("chainID", cfg.ID.ChainID, "id", cfg.ID)
	return &rpcL1ELNode{
		rpcELNode: newRpcELNode(cfg.ELNodeConfig),
		id:        cfg.ID,
	}
}

func (r *rpcL1ELNode) ID() L1ELNodeID {
	return r.id
}
