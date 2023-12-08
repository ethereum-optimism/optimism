//go:build !rethdb

package sources

import (
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/sources/caching"
	"github.com/ethereum/go-ethereum/log"
)

const buildRethdb = false

func newRecProviderFromConfig(client client.RPC, log log.Logger, metrics caching.Metrics, config *EthClientConfig) *CachingReceiptsProvider {
	return newRPCRecProviderFromConfig(client, log, metrics, config)
}
