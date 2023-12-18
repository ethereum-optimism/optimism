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

// ActiveL2EndpointProvider is an interface for providing a RollupClient
// It manages the lifecycle of the RollupClient for callers
// It does this by failing over down the list of rollupUrls if the current one is inactive or broken
type ActiveL2RollupProvider struct {
	checkDuration  time.Duration
	networkTimeout time.Duration
	log            log.Logger

	activeTimeout time.Time

	rollupUrls          []string
	rollupDialer        rollupDialer
	currentRollupClient RollupClientInterface
	rollupIndex         int
	clientLock          *sync.Mutex
}

// NewActiveL2RollupProvider creates a new ActiveL2RollupProvider
// the checkDuration is the duration between checks to see if the current rollup client is active
// provide a checkDuration of 0 to check every time
func NewActiveL2RollupProvider(
	ctx context.Context,
	rollupUrls []string,
	checkDuration time.Duration,
	networkTimeout time.Duration,
	logger log.Logger,
) (*ActiveL2RollupProvider, error) {
	rollupDialer := func(ctx context.Context, timeout time.Duration,
		log log.Logger, url string) (RollupClientInterface, error) {
		return DialRollupClientWithTimeout(ctx, timeout, log, url)
	}
	return newActiveL2RollupProvider(ctx, rollupUrls, checkDuration, networkTimeout, logger, rollupDialer)
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
	p := &ActiveL2RollupProvider{
		checkDuration:  checkDuration,
		networkTimeout: networkTimeout,
		log:            logger,
		rollupUrls:     rollupUrls,
		rollupDialer:   dialer,
		clientLock:     &sync.Mutex{},
		rollupIndex:    -1,
	}
	cctx, cancel := context.WithTimeout(ctx, networkTimeout)
	defer cancel()
	err := p.ensureClientInitialized(cctx)
	if err != nil {
		return nil, fmt.Errorf("dialing initial rollup client: %w", err)
	}
	_, err = p.RollupClient(cctx)
	if err != nil {
		return nil, fmt.Errorf("setting provider rollup client: %w", err)
	}
	return p, nil
}

var errSeqUnset = errors.New("sequencer unset")

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

func (p *ActiveL2RollupProvider) ensureClientInitialized(ctx context.Context) error {
	if p.currentRollupClient != nil {
		return nil
	}
	return p.dialNextSequencer(ctx)
}

func (p *ActiveL2RollupProvider) findActiveEndpoints(ctx context.Context) error {
	for range p.rollupUrls {
		active, err := p.checkCurrentSequencer(ctx)
		if errors.Is(err, errSeqUnset) {
			log.Debug("Current sequencer unset.")
		} else if ep := p.rollupUrls[p.rollupIndex]; err != nil {
			p.log.Warn("Error querying active sequencer, closing connection and trying next.", "err", err, "index", p.rollupIndex, "url", ep)
		} else if active {
			p.log.Debug("Current sequencer active.", "index", p.rollupIndex, "url", ep)
			return nil
		} else {
			p.log.Info("Current sequencer inactive, closing connection and trying next.", "index", p.rollupIndex, "url", ep)
		}
		if err := p.dialNextSequencer(ctx); err != nil {
			p.log.Warn("Error dialing next sequencer.", "err", err, "index", p.rollupIndex)
		}
	}
	return fmt.Errorf("failed to find an active sequencer, tried following urls: %v", p.rollupUrls)
}

func (p *ActiveL2RollupProvider) checkCurrentSequencer(ctx context.Context) (bool, error) {
	if p.currentRollupClient == nil {
		return false, errSeqUnset
	}
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

	p.rollupIndex = (p.rollupIndex + 1) % p.numEndpoints()
	ep := p.rollupUrls[p.rollupIndex]
	p.log.Info("Dialing next sequencer.", "index", p.rollupIndex, "url", ep)
	rollupClient, err := p.rollupDialer(cctx, p.networkTimeout, p.log, ep)
	if err != nil {
		return fmt.Errorf("dialing rollup client: %w", err)
	}
	if p.currentRollupClient != nil {
		p.currentRollupClient.Close()
	}
	p.currentRollupClient = rollupClient
	return nil
}

func (p *ActiveL2RollupProvider) Close() {
	if p.currentRollupClient != nil {
		p.currentRollupClient.Close()
	}
}
