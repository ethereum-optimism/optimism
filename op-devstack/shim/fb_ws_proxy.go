package shim

import (
	"net/http"

	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type FlashblocksWebsocketProxyConfig struct {
	CommonConfig
	ID        stack.FlashblocksWebsocketProxyID
	WsUrl     string
	WsHeaders http.Header
}

type flashblocksWebsocketProxy struct {
	commonImpl
	id        stack.FlashblocksWebsocketProxyID
	wsUrl     string
	wsHeaders http.Header
}

var _ stack.FlashblocksWebsocketProxy = (*flashblocksWebsocketProxy)(nil)

func NewFlashblocksWebsocketProxy(cfg FlashblocksWebsocketProxyConfig) stack.FlashblocksWebsocketProxy {
	cfg.T = cfg.T.WithCtx(stack.ContextWithID(cfg.T.Ctx(), cfg.ID))
	return &flashblocksWebsocketProxy{
		commonImpl: newCommon(cfg.CommonConfig),
		id:         cfg.ID,
		wsUrl:      cfg.WsUrl,
		wsHeaders:  cfg.WsHeaders,
	}
}

func (r *flashblocksWebsocketProxy) ID() stack.FlashblocksWebsocketProxyID {
	return r.id
}

func (r *flashblocksWebsocketProxy) ChainID() eth.ChainID {
	return r.id.ChainID()
}

func (r *flashblocksWebsocketProxy) WsUrl() string {
	return r.wsUrl
}

func (r *flashblocksWebsocketProxy) WsHeaders() http.Header {
	return r.wsHeaders
}
