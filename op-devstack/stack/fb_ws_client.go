package stack

import (
	"net/http"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type FlashblocksWSClient interface {
	Common
	ChainID() eth.ChainID
	ID() ComponentID
	WsUrl() string
	WsHeaders() http.Header
}
