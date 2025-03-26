package fetch

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

type Fetcher struct {
	L1RPCURL              string
	SystemConfigProxy     common.Address
	L1StandardBridgeProxy common.Address
	lgr                   log.Logger
}

func NewFetcher(lgr log.Logger, l1RPCURL string, systemConfigProxy, l1StandardBridge common.Address) (*Fetcher, error) {
	return &Fetcher{
		L1RPCURL:              l1RPCURL,
		SystemConfigProxy:     systemConfigProxy,
		L1StandardBridgeProxy: l1StandardBridge,
		lgr:                   lgr,
	}, nil
}
