package stack

import (
	"log/slog"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type FlashblocksWebsocketProxy interface {
	Common
	ChainID() eth.ChainID
	ID() FlashblocksWebsocketProxyID
	WsUrl() string
}

type FlashblocksWebsocketProxyID idWithChain

const FlashblocksWebsocketProxyKind Kind = "FlashblocksWebsocketProxy"

func NewFlashblocksWebsocketProxyID(key string, chainID eth.ChainID) FlashblocksWebsocketProxyID {
	return FlashblocksWebsocketProxyID{
		key:     key,
		chainID: chainID,
	}
}

func (id FlashblocksWebsocketProxyID) String() string {
	return idWithChain(id).string(FlashblocksWebsocketProxyKind)
}

func (id FlashblocksWebsocketProxyID) ChainID() eth.ChainID {
	return idWithChain(id).chainID
}

func (id FlashblocksWebsocketProxyID) MarshalText() ([]byte, error) {
	return idWithChain(id).marshalText(FlashblocksWebsocketProxyKind)
}

func (id FlashblocksWebsocketProxyID) LogValue() slog.Value {
	return slog.StringValue(id.String())
}

func (id *FlashblocksWebsocketProxyID) UnmarshalText(data []byte) error {
	return (*idWithChain)(id).unmarshalText(FlashblocksWebsocketProxyKind, data)
}

func SortFlashblocksWebsocketProxyIDs(ids []FlashblocksWebsocketProxyID) []FlashblocksWebsocketProxyID {
	return copyAndSort(ids, func(a, b FlashblocksWebsocketProxyID) bool {
		return lessIDWithChain(idWithChain(a), idWithChain(b))
	})
}

func SortFlashblocksWebsocketProxies(elems []FlashblocksWebsocketProxy) []FlashblocksWebsocketProxy {
	return copyAndSort(elems, func(a, b FlashblocksWebsocketProxy) bool {
		return lessIDWithChain(idWithChain(a.ID()), idWithChain(b.ID()))
	})
}

var _ FlashblocksBuilderMatcher = FlashblocksBuilderID{}

func (id FlashblocksWebsocketProxyID) Match(elems []FlashblocksWebsocketProxy) []FlashblocksWebsocketProxy {
	return findByID(id, elems)
}
