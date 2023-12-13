package dial

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/log"
)

const DefaultActiveSequencerFollowerCheckDuration = 2 * DefaultDialTimeout

type ActiveL2EndpointProvider struct {
	ActiveL2RollupProvider
	ethClients []EthClientInterface
}

func NewActiveL2EndpointProvider(
	ctx context.Context,
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

	rollupProvider, err := NewActiveL2RollupProvider(ctx, rollupUrls, checkDuration, networkTimeout, logger)
	if err != nil {
		return nil, err
	}
	cctx, cancel := context.WithTimeout(ctx, networkTimeout)
	defer cancel()
	ethClients := make([]EthClientInterface, 0, len(ethUrls))
	for _, url := range ethUrls {

		ethClient, err := DialEthClientWithTimeout(cctx, networkTimeout, logger, url)
		if err != nil {
			return nil, fmt.Errorf("dialing eth client: %w", err)
		}
		ethClients = append(ethClients, ethClient)
	}

	return &ActiveL2EndpointProvider{
		ActiveL2RollupProvider: *rollupProvider,
		ethClients:             ethClients,
	}, nil
}

func (p *ActiveL2EndpointProvider) EthClient(ctx context.Context) (EthClientInterface, error) {
	err := p.ensureActiveEndpoint(ctx)
	if err != nil {
		return nil, err
	}
	p.clientLock.Lock()
	defer p.clientLock.Unlock()
	return p.ethClients[p.currentIdx], nil
}

func (p *ActiveL2EndpointProvider) Close() {
	for _, ethClient := range p.ethClients {
		ethClient.Close()
	}
	p.ActiveL2RollupProvider.Close()
}
