package dial

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
)

type ActiveL2EndpointProvider struct {
	ActiveL2RollupProvider
	ethEndpoints     []string
	currentEthClient *ethclient.Client
}

func NewActiveL2EndpointProvider(
	ethUrls, rollupUrls []string,
	checkDuration time.Duration,
	networkTimeout time.Duration,
	logger log.Logger,
) (*ActiveL2EndpointProvider, error) {
	if len(rollupUrls) == 0 {
		return nil, errors.New("empty rollup urls list")
	}
	if len(ethUrls) != len(rollupUrls) {
		return nil, errors.New("number of eth urls and rollup urls mismatch")
	}

	rollupProvider, err := NewActiveL2RollupProvider(rollupUrls, checkDuration, networkTimeout, logger)
	if err != nil {
		return nil, err
	}

	return &ActiveL2EndpointProvider{
		ActiveL2RollupProvider: *rollupProvider,
		ethEndpoints:           ethUrls,
	}, nil
}

func (p *ActiveL2EndpointProvider) EthClient(ctx context.Context) (*ethclient.Client, error) {
	err := p.ensureActiveEndpoint(ctx)
	if err != nil {
		return nil, err
	}
	p.clientLock.Lock()
	defer p.clientLock.Unlock()
	return p.currentEthClient, nil
}

func (p *ActiveL2EndpointProvider) RollupClient(ctx context.Context) (*sources.RollupClient, error) {
	return p.ActiveL2RollupProvider.RollupClient(ctx)
}

func (p *ActiveL2EndpointProvider) ensureActiveEndpoint(ctx context.Context) error {
	return p.ActiveL2RollupProvider.ensureActiveEndpoint(ctx)
}

func (p *ActiveL2EndpointProvider) shouldCheck() bool {
	return p.ActiveL2RollupProvider.shouldCheck()
}

func (p *ActiveL2EndpointProvider) findActiveEndpoints(ctx context.Context) error {
	return p.ActiveL2RollupProvider.findActiveEndpoints(ctx)
}

func (p *ActiveL2EndpointProvider) checkCurrentSequencer(ctx context.Context) (bool, error) {
	return p.ActiveL2RollupProvider.checkCurrentSequencer(ctx)
}

func (p *ActiveL2EndpointProvider) dialNextSequencer(ctx context.Context, idx int) error {
	cctx, cancel := context.WithTimeout(ctx, p.networkTimeout)
	defer cancel()

	ethClient, err := DialEthClientWithTimeout(cctx, p.networkTimeout, p.log, p.ethEndpoints[idx])
	if err != nil {
		return fmt.Errorf("dialing eth client: %w", err)
	}

	rollupClient, err := DialRollupClientWithTimeout(cctx, p.networkTimeout, p.log, p.rollupEndpoints[idx])
	if err != nil {
		return fmt.Errorf("dialing rollup client: %w", err)
	}
	p.clientLock.Lock()
	defer p.clientLock.Unlock()
	p.currentEthClient, p.currentRollupClient = ethClient, rollupClient
	return nil
}

func (p *ActiveL2EndpointProvider) NumEndpoints() int {
	return len(p.ethEndpoints)
}

func (p *ActiveL2EndpointProvider) Close() {
	if p.currentEthClient != nil {
		p.currentEthClient.Close()
	}
	p.ActiveL2RollupProvider.Close()
}
