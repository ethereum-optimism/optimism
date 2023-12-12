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

const DefaultActiveSequencerFollowerCheckDuration = 2 * DefaultDialTimeout

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
	err := p.ensureActiveEndpoint(ctx)
	if err != nil {
		return nil, err
	}
	p.clientLock.Lock()
	defer p.clientLock.Unlock()
	return p.currentRollupClient, nil
}

func (p *ActiveL2EndpointProvider) ensureActiveEndpoint(ctx context.Context) error {
	if !p.shouldCheck() {
		return nil
	}

	if err := p.findActiveEndpoints(ctx); err != nil {
		return err
	}
	p.activeTimeout = time.Now().Add(p.checkDuration)
	return nil
}

func (p *ActiveL2EndpointProvider) findActiveEndpoints(ctx context.Context) error {
	// If current is not active, dial new sequencers until finding an active one.
	ts := time.Now()
	for i := 0; ; i++ {
		active, err := p.checkCurrentSequencer(ctx)
		if err != nil {
			if ctx.Err() != nil {
				p.log.Warn("Error querying active sequencer, trying next.", "err", err, "try", i)
				return fmt.Errorf("querying active sequencer: %w", err)
			}
			p.log.Warn("Error querying active sequencer, trying next.", "err", err, "try", i)
		} else if active {
			p.log.Debug("Current sequencer active.", "try", i)
			return nil
		} else {
			p.log.Info("Current sequencer inactive, trying next.", "try", i)
		}

		// After iterating over all endpoints, sleep if all were just inactive,
		// to avoid spamming the sequencers in a loop.
		if (i+1)%p.NumEndpoints() == 0 {
			d := time.Until(ts.Add(p.checkDuration))
			time.Sleep(d) // accepts negative
			ts = time.Now()
		}

		if err := p.dialNextSequencer(ctx, i); err != nil {
			return fmt.Errorf("dialing next sequencer: %w", err)
		}
	}
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

func (p *ActiveL2EndpointProvider) Close() {
	if p.currentEthClient != nil {
		p.currentEthClient.Close()
	}
	p.ActiveL2RollupProvider.Close()
}
