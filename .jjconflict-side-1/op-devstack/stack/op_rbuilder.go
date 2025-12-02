package stack

import (
	"log/slog"

	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// OPRBuilderNodeID identifies a L2ELNode by name and chainID, is type-safe, and can be value-copied and used as map key.
type OPRBuilderNodeID idWithChain

var _ IDWithChain = (*OPRBuilderNodeID)(nil)

const OPRBuilderNodeKind Kind = "OPRBuilderNode"

func NewOPRBuilderNodeID(key string, chainID eth.ChainID) OPRBuilderNodeID {
	return OPRBuilderNodeID{
		key:     key,
		chainID: chainID,
	}
}

func (id OPRBuilderNodeID) String() string {
	return idWithChain(id).string(OPRBuilderNodeKind)
}

func (id OPRBuilderNodeID) ChainID() eth.ChainID {
	return id.chainID
}

func (id OPRBuilderNodeID) Kind() Kind {
	return OPRBuilderNodeKind
}

func (id OPRBuilderNodeID) Key() string {
	return id.key
}

func (id OPRBuilderNodeID) LogValue() slog.Value {
	return slog.StringValue(id.String())
}

func (id OPRBuilderNodeID) MarshalText() ([]byte, error) {
	return idWithChain(id).marshalText(OPRBuilderNodeKind)
}

func (id *OPRBuilderNodeID) UnmarshalText(data []byte) error {
	return (*idWithChain)(id).unmarshalText(OPRBuilderNodeKind, data)
}

func SortOPRBuilderIDs(ids []OPRBuilderNodeID) []OPRBuilderNodeID {
	return copyAndSort(ids, func(a, b OPRBuilderNodeID) bool {
		return lessIDWithChain(idWithChain(a), idWithChain(b))
	})
}

func SortOPRBuilderNodes(elems []OPRBuilderNode) []OPRBuilderNode {
	return copyAndSort(elems, func(a, b OPRBuilderNode) bool {
		return lessIDWithChain(idWithChain(a.ID()), idWithChain(b.ID()))
	})
}

var _ OPRBuilderNodeMatcher = OPRBuilderNodeID{}

func (id OPRBuilderNodeID) Match(elems []OPRBuilderNode) []OPRBuilderNode {
	return findByID(id, elems)
}

// L2ELNode is a L2 ethereum execution-layer node
type OPRBuilderNode interface {
	ID() OPRBuilderNodeID
	L2EthClient() apis.L2EthClient
	L2EngineClient() apis.EngineClient
	FlashblocksClient() *client.WSClient

	ELNode
}
