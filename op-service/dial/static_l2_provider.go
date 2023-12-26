package dial

import (
	"context"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
)

// L2EndpointProvider is an interface for providing a RollupClient and l2 eth client
// It manages the lifecycle of the RollupClient and eth client for callers
// It does this by extending the RollupProvider interface to add the ability to get an EthClient
type L2EndpointProvider interface {
	RollupProvider
	// EthClient(ctx) returns the underlying ethclient pointing to the L2 execution node
	EthClient(ctx context.Context) (EthClientInterface, error)
}

// StaticL2EndpointProvider is a L2EndpointProvider that always returns the same static RollupClient and eth client
// It is meant for scenarios where a single, unchanging (L2 rollup node, L2 execution node) pair is used
type StaticL2EndpointProvider struct {
	StaticL2RollupProvider
	ethClient *ethclient.Client
}

func NewStaticL2EndpointProvider(ctx context.Context, log log.Logger, ethClientUrl string, rollupClientUrl string) (*StaticL2EndpointProvider, error) {
	ethClient, err := DialEthClientWithTimeout(ctx, DefaultDialTimeout, log, ethClientUrl)
	if err != nil {
		return nil, err
	}
	rollupProvider, err := NewStaticL2RollupProvider(ctx, log, rollupClientUrl)
	if err != nil {
		return nil, err
	}
	return &StaticL2EndpointProvider{
		StaticL2RollupProvider: *rollupProvider,
		ethClient:              ethClient,
	}, nil
}

func (p *StaticL2EndpointProvider) EthClient(context.Context) (EthClientInterface, error) {
	return p.ethClient, nil
}

func (p *StaticL2EndpointProvider) Close() {
	if p.ethClient != nil {
		p.ethClient.Close()
	}
	p.StaticL2RollupProvider.Close()
}
