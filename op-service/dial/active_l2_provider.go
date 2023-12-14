package dial

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/log"
)

const DefaultActiveSequencerFollowerCheckDuration = 2 * DefaultDialTimeout

type ethDialer func(ctx context.Context, timeout time.Duration, log log.Logger, url string) (EthClientInterface, error)

type ActiveL2EndpointProvider struct {
	ActiveL2RollupProvider
	currentEthClient EthClientInterface
	ethDialer        ethDialer
	ethUrls          []string
}

func NewActiveL2EndpointProvider(ctx context.Context, ethUrls, rollupUrls []string, checkDuration time.Duration, networkTimeout time.Duration, logger log.Logger) (*ActiveL2EndpointProvider, error) {
	dialEthClientInterfaceWithTimeout := func(ctx context.Context, timeout time.Duration, log log.Logger, url string) (EthClientInterface, error) {
		return DialEthClientWithTimeout(ctx, timeout, log, url)
	}
	dialRollupClientInterfaceWithTimeout := func(ctx context.Context, timeout time.Duration, log log.Logger, url string) (RollupClientInterface, error) {
		return DialRollupClientWithTimeout(ctx, timeout, log, url)
	}
	return newActiveL2EndpointProvider(ctx, ethUrls, rollupUrls, checkDuration, networkTimeout, logger, dialEthClientInterfaceWithTimeout, dialRollupClientInterfaceWithTimeout)
}

func newActiveL2EndpointProvider(
	ctx context.Context,
	ethUrls, rollupUrls []string,
	checkDuration time.Duration,
	networkTimeout time.Duration,
	logger log.Logger,
	ethDialer ethDialer,
	rollupDialer rollupDialer,
) (*ActiveL2EndpointProvider, error) {
	if len(rollupUrls) == 0 {
		return nil, errors.New("empty rollup urls list")
	}
	if len(ethUrls) != len(rollupUrls) {
		return nil, errors.New("number of eth urls and rollup urls mismatch")
	}

	rollupProvider, err := newActiveL2RollupProvider(ctx, rollupUrls, checkDuration, networkTimeout, logger, rollupDialer)
	if err != nil {
		return nil, err
	}
	cctx, cancel := context.WithTimeout(ctx, networkTimeout)
	defer cancel()
	ethClient, err := ethDialer(cctx, networkTimeout, logger, ethUrls[0])
	if err != nil {
		return nil, fmt.Errorf("dialing eth client: %w", err)
	}
	return &ActiveL2EndpointProvider{
		ActiveL2RollupProvider: *rollupProvider,
		currentEthClient:       ethClient,
		ethDialer:              ethDialer,
		ethUrls:                ethUrls,
	}, nil
}

func (p *ActiveL2EndpointProvider) EthClient(ctx context.Context) (EthClientInterface, error) {
	p.clientLock.Lock()
	defer p.clientLock.Unlock()
	indexBeforeCheck := p.currentIndex
	err := p.ensureActiveEndpoint(ctx)
	if err != nil {
		return nil, err
	}
	if indexBeforeCheck != p.currentIndex {
		// we changed sequencers, dial a new EthClient
		cctx, cancel := context.WithTimeout(ctx, p.networkTimeout)
		defer cancel()
		ep := p.ethUrls[p.currentIndex]
		log.Info("sequencer changed, dialing new eth client", "new_index", p.currentIndex, "new_url", ep)
		ethClient, err := p.ethDialer(cctx, p.networkTimeout, p.log, ep)
		if err != nil {
			return nil, fmt.Errorf("dialing eth client: %w", err)
		}
		p.currentEthClient.Close()
		p.currentEthClient = ethClient
	}
	return p.currentEthClient, nil
}

func (p *ActiveL2EndpointProvider) Close() {
	p.currentEthClient.Close()
	p.ActiveL2RollupProvider.Close()
}
