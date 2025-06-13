package stack

import (
	"log/slog"

	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type FlashblocksBuilderNode interface {
	ELNode
	ID() FlashblocksBuilderID
	ConductorID() ConductorID
	L2EthClient() apis.L2EthClient
	FlashblocksWsUrl() string
}

type FlashblocksBuilderID idWithChain

const FlashblocksBuilderKind Kind = "FlashblocksBuilder"

func NewFlashblocksBuilderID(key string, chainID eth.ChainID) FlashblocksBuilderID {
	return FlashblocksBuilderID{
		key:     key,
		chainID: chainID,
	}
}

func (id FlashblocksBuilderID) String() string {
	return idWithChain(id).string(FlashblocksBuilderKind)
}

func (id FlashblocksBuilderID) ChainID() eth.ChainID {
	return idWithChain(id).chainID
}

func (id FlashblocksBuilderID) MarshalText() ([]byte, error) {
	return idWithChain(id).marshalText(FlashblocksBuilderKind)
}

func (id FlashblocksBuilderID) LogValue() slog.Value {
	return slog.StringValue(id.String())
}

func (id *FlashblocksBuilderID) UnmarshalText(data []byte) error {
	return (*idWithChain)(id).unmarshalText(FlashblocksBuilderKind, data)
}

func SortFlashblocksBuilderIDs(ids []FlashblocksBuilderID) []FlashblocksBuilderID {
	return copyAndSort(ids, func(a, b FlashblocksBuilderID) bool {
		return lessIDWithChain(idWithChain(a), idWithChain(b))
	})
}

func SortFlashblocksBuilders(elems []FlashblocksBuilderNode) []FlashblocksBuilderNode {
	return copyAndSort(elems, func(a, b FlashblocksBuilderNode) bool {
		return lessIDWithChain(idWithChain(a.ID()), idWithChain(b.ID()))
	})
}

var _ FlashblocksBuilderMatcher = FlashblocksBuilderID{}

func (id FlashblocksBuilderID) Match(elems []FlashblocksBuilderNode) []FlashblocksBuilderNode {
	return findByID(id, elems)
}
