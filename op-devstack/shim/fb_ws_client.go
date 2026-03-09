package shim

import (
	"net/http"

	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type FlashblocksWSClientConfig struct {
	CommonConfig
	ID        stack.ComponentID
	WsUrl     string
	WsHeaders http.Header
}

type flashblocksWSClient struct {
	commonImpl
	id        stack.ComponentID
	wsUrl     string
	wsHeaders http.Header
}

var _ stack.FlashblocksWSClient = (*flashblocksWSClient)(nil)

func NewFlashblocksWSClient(cfg FlashblocksWSClientConfig) stack.FlashblocksWSClient {
	cfg.T = cfg.T.WithCtx(stack.ContextWithID(cfg.T.Ctx(), cfg.ID))
	return &flashblocksWSClient{
		commonImpl: newCommon(cfg.CommonConfig),
		id:         cfg.ID,
		wsUrl:      cfg.WsUrl,
		wsHeaders:  cfg.WsHeaders,
	}
}

func (r *flashblocksWSClient) ID() stack.ComponentID {
	return r.id
}

func (r *flashblocksWSClient) ChainID() eth.ChainID {
	return r.id.ChainID()
}

func (r *flashblocksWSClient) WsUrl() string {
	return r.wsUrl
}

func (r *flashblocksWSClient) WsHeaders() http.Header {
	return r.wsHeaders
}
