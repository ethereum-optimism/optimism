package dial

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/log"
)

type rollupDialer func(ctx context.Context, timeout time.Duration, log log.Logger, url string) (RollupClientInterface, error)

type ActiveL2RollupProvider struct {
	checkDuration  time.Duration
	networkTimeout time.Duration
	log            log.Logger

	activeTimeout time.Time

	rollupUrls          []string
	rollupDialer        rollupDialer
	currentRollupClient RollupClientInterface
	currentIndex        int
	clientLock          *sync.Mutex
}

func NewActiveL2RollupProvider(
	ctx context.Context,
	rollupUrls []string,
	checkDuration time.Duration,
	networkTimeout time.Duration,
	logger log.Logger,
) (*ActiveL2RollupProvider, error) {
	dialRollupClientInterfaceWithTimeout := func(ctx context.Context, timeout time.Duration, log log.Logger, url string) (RollupClientInterface, error) {
		return DialRollupClientWithTimeout(ctx, timeout, log, url)
	}
	return newActiveL2RollupProvider(ctx, rollupUrls, checkDuration, networkTimeout, logger, dialRollupClientInterfaceWithTimeout)
}

func newActiveL2RollupProvider(
	ctx context.Context,
	rollupUrls []string,
	checkDuration time.Duration,
	networkTimeout time.Duration,
	logger log.Logger,
	dialer rollupDialer,
) (*ActiveL2RollupProvider, error) {
	if len(rollupUrls) == 0 {
		return nil, errors.New("empty rollup urls list")
	}

	cctx, cancel := context.WithTimeout(ctx, networkTimeout)
	defer cancel()

	rollupClient, err := dialer(cctx, networkTimeout, logger, rollupUrls[0])
	if err != nil {
		return nil, fmt.Errorf("dialing rollup client: %w", err)
	}

	return &ActiveL2RollupProvider{
		checkDuration:       checkDuration,
		networkTimeout:      networkTimeout,
		log:                 logger,
		rollupUrls:          rollupUrls,
		rollupDialer:        dialer,
		currentRollupClient: rollupClient,
		clientLock:          &sync.Mutex{},
	}, nil
}

func (p *ActiveL2RollupProvider) RollupClient(ctx context.Context) (RollupClientInterface, error) {
	p.clientLock.Lock()
	defer p.clientLock.Unlock()
	err := p.ensureActiveEndpoint(ctx)
	if err != nil {
		return nil, err
	}
	return p.currentRollupClient, nil
}

func (p *ActiveL2RollupProvider) ensureActiveEndpoint(ctx context.Context) error {
	if !p.shouldCheck() {
		return nil
	}

	if err := p.findActiveEndpoints(ctx); err != nil {
		return err
	}
	p.activeTimeout = time.Now().Add(p.checkDuration)
	return nil
}

func (p *ActiveL2RollupProvider) shouldCheck() bool {
	return time.Now().After(p.activeTimeout)
}

func (p *ActiveL2RollupProvider) findActiveEndpoints(ctx context.Context) error {
	for attempt := range p.rollupUrls {
		active, err := p.checkCurrentSequencer(ctx)
		if err != nil {
			p.log.Warn("Error querying active sequencer, closing connection and trying next.", "err", err, "try", attempt)
			p.currentRollupClient.Close()
		} else if active {
			p.log.Debug("Current sequencer active.", "try", attempt)
			return nil
		} else {
			p.log.Info("Current sequencer inactive, closing connection and trying next.", "try", attempt)
			p.currentRollupClient.Close()
		}
		if err := p.dialNextSequencer(ctx); err != nil {
			return fmt.Errorf("dialing next sequencer: %w", err)
		}
	}
	return fmt.Errorf("failed to find an active sequencer, tried following urls: %v", p.rollupUrls)
}

func (p *ActiveL2RollupProvider) checkCurrentSequencer(ctx context.Context) (bool, error) {
	cctx, cancel := context.WithTimeout(ctx, p.networkTimeout)
	defer cancel()
	return p.currentRollupClient.SequencerActive(cctx)
}

func (p *ActiveL2RollupProvider) numEndpoints() int {
	return len(p.rollupUrls)
}

func (p *ActiveL2RollupProvider) dialNextSequencer(ctx context.Context) error {
	cctx, cancel := context.WithTimeout(ctx, p.networkTimeout)
	defer cancel()

	p.currentIndex = (p.currentIndex + 1) % p.numEndpoints()
	ep := p.rollupUrls[p.currentIndex]

	rollupClient, err := p.rollupDialer(cctx, p.networkTimeout, p.log, ep)
	if err != nil {
		return fmt.Errorf("dialing rollup client: %w", err)
	}
	p.currentRollupClient = rollupClient
	return nil
}

func (p *ActiveL2RollupProvider) Close() {
	p.currentRollupClient.Close()
}
