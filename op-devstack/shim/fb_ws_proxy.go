package shim

import (
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type FlashblocksWebsocketProxyConfig struct {
	CommonConfig
	ID    stack.FlashblocksWebsocketProxyID
	WsUrl string
}

type flashblocksWebsocketProxy struct {
	commonImpl
	id    stack.FlashblocksWebsocketProxyID
	wsUrl string
}

var _ stack.FlashblocksWebsocketProxy = (*flashblocksWebsocketProxy)(nil)

func NewFlashblocksWebsocketProxy(cfg FlashblocksWebsocketProxyConfig) stack.FlashblocksWebsocketProxy {
	cfg.T = cfg.T.WithCtx(stack.ContextWithID(cfg.T.Ctx(), cfg.ID))
	return &flashblocksWebsocketProxy{
		commonImpl: newCommon(cfg.CommonConfig),
		id:         cfg.ID,
		wsUrl:      cfg.WsUrl,
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
