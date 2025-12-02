package stack

import (
	"log/slog"
	"net/http"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type FlashblocksWSClient interface {
	Common
	ChainID() eth.ChainID
	ID() FlashblocksWSClientID
	WsUrl() string
	WsHeaders() http.Header
}

type FlashblocksWSClientID idWithChain

const FlashblocksWSClientKind Kind = "FlashblocksWSClient"

func NewFlashblocksWSClientID(key string, chainID eth.ChainID) FlashblocksWSClientID {
	return FlashblocksWSClientID{
		key:     key,
		chainID: chainID,
	}
}

func (id FlashblocksWSClientID) String() string {
	return idWithChain(id).string(FlashblocksWSClientKind)
}

func (id FlashblocksWSClientID) ChainID() eth.ChainID {
	return idWithChain(id).chainID
}

func (id FlashblocksWSClientID) MarshalText() ([]byte, error) {
	return idWithChain(id).marshalText(FlashblocksWSClientKind)
}

func (id FlashblocksWSClientID) LogValue() slog.Value {
	return slog.StringValue(id.String())
}

func (id *FlashblocksWSClientID) UnmarshalText(data []byte) error {
	return (*idWithChain)(id).unmarshalText(FlashblocksWSClientKind, data)
}

func SortFlashblocksWSClientIDs(ids []FlashblocksWSClientID) []FlashblocksWSClientID {
	return copyAndSort(ids, func(a, b FlashblocksWSClientID) bool {
		return lessIDWithChain(idWithChain(a), idWithChain(b))
	})
}

func SortFlashblocksWSClients(elems []FlashblocksWSClient) []FlashblocksWSClient {
	return copyAndSort(elems, func(a, b FlashblocksWSClient) bool {
		return lessIDWithChain(idWithChain(a.ID()), idWithChain(b.ID()))
	})
}

func (id FlashblocksWSClientID) Match(elems []FlashblocksWSClient) []FlashblocksWSClient {
	return findByID(id, elems)
}
