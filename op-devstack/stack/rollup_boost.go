package stack

import (
	"log/slog"

	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// RollupBoostNodeID identifies a RollupBoost node by name and chainID, is type-safe, and can be value-copied and used as map key.
type RollupBoostNodeID L2ELNodeID

var _ IDWithChain = (*RollupBoostNodeID)(nil)

const RollupBoostNodeKind Kind = "RollupBoostNode"

func NewRollupBoostNodeID(key string, chainID eth.ChainID) RollupBoostNodeID {
	return RollupBoostNodeID{
		key:     key,
		chainID: chainID,
	}
}

func (id RollupBoostNodeID) String() string {
	return idWithChain(id).string(RollupBoostNodeKind)
}

func (id RollupBoostNodeID) ChainID() eth.ChainID {
	return id.chainID
}

func (id RollupBoostNodeID) Kind() Kind {
	return RollupBoostNodeKind
}

func (id RollupBoostNodeID) Key() string {
	return id.key
}

func (id RollupBoostNodeID) LogValue() slog.Value {
	return slog.StringValue(id.String())
}

func (id RollupBoostNodeID) MarshalText() ([]byte, error) {
	return idWithChain(id).marshalText(RollupBoostNodeKind)
}

func (id *RollupBoostNodeID) UnmarshalText(data []byte) error {
	return (*idWithChain)(id).unmarshalText(RollupBoostNodeKind, data)
}

func SortRollupBoostIDs(ids []RollupBoostNodeID) []RollupBoostNodeID {
	return copyAndSort(ids, func(a, b RollupBoostNodeID) bool {
		return lessIDWithChain(idWithChain(a), idWithChain(b))
	})
}

func SortRollupBoostNodes(elems []RollupBoostNode) []RollupBoostNode {
	return copyAndSort(elems, func(a, b RollupBoostNode) bool {
		return lessIDWithChain(idWithChain(a.ID()), idWithChain(b.ID()))
	})
}

var _ RollupBoostNodeMatcher = RollupBoostNodeID{}

func (id RollupBoostNodeID) Match(elems []RollupBoostNode) []RollupBoostNode {
	return findByID(id, elems)
}

// RollupBoostNode is a shim service between an L2 consensus-layer node and an L2 ethereum execution-layer node
type RollupBoostNode interface {
	ID() RollupBoostNodeID
	L2EthClient() apis.L2EthClient
	L2EngineClient() apis.EngineClient
	FlashblocksClient() *client.WSClient

	ELNode
}
