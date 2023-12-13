package dial

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/log"
)

type ActiveL2RollupProvider struct {
	checkDuration  time.Duration
	networkTimeout time.Duration
	log            log.Logger

	activeTimeout time.Time

	currentIdx    int
	rollupClients []RollupClientInterface
	clientLock    *sync.Mutex
}

func NewActiveL2RollupProvider(
	ctx context.Context,
	rollupUrls []string,
	checkDuration time.Duration,
	networkTimeout time.Duration,
	logger log.Logger,
) (*ActiveL2RollupProvider, error) {
	if len(rollupUrls) == 0 {
		return nil, errors.New("empty rollup urls list")
	}

	cctx, cancel := context.WithTimeout(ctx, networkTimeout)
	defer cancel()

	rollupClients := make([]RollupClientInterface, 0, len(rollupUrls))
	for _, url := range rollupUrls {
		rollupClient, err := DialRollupClientWithTimeout(cctx, networkTimeout, logger, url)
		if err != nil {
			return nil, fmt.Errorf("dialing rollup client: %w", err)
		}
		rollupClients = append(rollupClients, rollupClient)
	}

	return &ActiveL2RollupProvider{
		checkDuration:  checkDuration,
		networkTimeout: networkTimeout,
		log:            logger,
		rollupClients:  rollupClients,
		clientLock:     &sync.Mutex{},
	}, nil
}

func (p *ActiveL2RollupProvider) RollupClient(ctx context.Context) (RollupClientInterface, error) {
	err := p.ensureActiveEndpoint(ctx)
	if err != nil {
		return nil, err
	}
	p.clientLock.Lock()
	defer p.clientLock.Unlock()
	return p.rollupClients[p.currentIdx], nil
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
	const maxRetries = 10
	ts := time.Now()
	for i := 0; i < maxRetries; i++ {
		active, err := p.checkSequencer(ctx, i%p.numEndpoints())
		if err != nil {
			p.log.Warn("Error querying active sequencer", "err", err, "try", i)
			if ctx.Err() != nil {
				return fmt.Errorf("querying active sequencer: %w", err)
			}
		} else if active {
			p.log.Debug("Current sequencer active", "index", i)
			p.currentIdx = i
			return nil
		} else {
			p.log.Info("Current sequencer inactive", "index", i)
		}

		if i%p.numEndpoints() == 0 {
			d := time.Until(ts.Add(p.checkDuration))
			time.Sleep(d) // Accepts negative duration
		}
	}
	return fmt.Errorf("failed to find an active sequencer after %d retries", maxRetries)
}

func (p *ActiveL2RollupProvider) checkSequencer(ctx context.Context, idx int) (bool, error) {
	cctx, cancel := context.WithTimeout(ctx, p.networkTimeout)
	defer cancel()
	active, err := p.rollupClients[idx].SequencerActive(cctx)
	p.log.Info("Checked whether sequencer is active", "index", idx, "active", active, "err", err)
	return active, err
}

func (p *ActiveL2RollupProvider) numEndpoints() int {
	return len(p.rollupClients)
}

func (p *ActiveL2RollupProvider) Close() {
	for _, client := range p.rollupClients {
		client.Close()
	}
}
