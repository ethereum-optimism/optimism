package sources

import (
	"github.com/ethereum-optimism/optimism/op-node/client"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/sources/caching"
	"github.com/ethereum/go-ethereum/log"
)

type SyncClient struct {
	*L2Client
}

type SyncClientConfig struct {
	L2ClientConfig
}

func SyncClientDefaultConfig(config *rollup.Config, trustRPC bool) *SyncClientConfig {
	return &SyncClientConfig{
		*L2ClientDefaultConfig(config, trustRPC),
	}
}

func NewSyncClient(client client.RPC, log log.Logger, metrics caching.Metrics, config *SyncClientConfig) (*SyncClient, error) {
	l2Client, err := NewL2Client(client, log, metrics, &config.L2ClientConfig)
	if err != nil {
		return nil, err
	}

	return &SyncClient{
		L2Client: l2Client,
	}, nil
}
