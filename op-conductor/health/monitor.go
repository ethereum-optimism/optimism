package health

import (
	"context"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/p2p"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/dial"
)

// HealthMonitor defines the interface for monitoring the health of the sequencer.
//
//go:generate mockery --name HealthMonitor --output mocks/ --with-expecter=true
type HealthMonitor interface {
	// Subscribe returns a channel that will be notified for every health check.
	Subscribe() <-chan bool
	// Start starts the health check.
	Start() error
	// Stop stops the health check.
	Stop() error
}

// NewSequencerHealthMonitor creates a new sequencer health monitor.
// interval is the interval between health checks measured in seconds.
// safeInterval is the interval between safe head progress measured in seconds.
// minPeerCount is the minimum number of peers required for the sequencer to be healthy.
func NewSequencerHealthMonitor(log log.Logger, interval, safeInterval, minPeerCount uint64, rollupCfg *rollup.Config, node dial.RollupClientInterface, p2p p2p.API) HealthMonitor {
	return &SequencerHealthMonitor{
		log:            log,
		done:           make(chan struct{}),
		interval:       interval,
		healthUpdateCh: make(chan bool),
		rollupCfg:      rollupCfg,
		safeInterval:   safeInterval,
		minPeerCount:   minPeerCount,
		node:           node,
		p2p:            p2p,
	}
}

// SequencerHealthMonitor monitors sequencer health.
type SequencerHealthMonitor struct {
	log  log.Logger
	done chan struct{}
	wg   sync.WaitGroup

	rollupCfg      *rollup.Config
	safeInterval   uint64
	minPeerCount   uint64
	interval       uint64
	healthUpdateCh chan bool

	node dial.RollupClientInterface
	p2p  p2p.API
}

var _ HealthMonitor = (*SequencerHealthMonitor)(nil)

// Start implements HealthMonitor.
func (hm *SequencerHealthMonitor) Start() error {
	hm.log.Info("starting health monitor")
	hm.wg.Add(1)
	go hm.loop()

	hm.log.Info("health monitor started")
	return nil
}

// Stop implements HealthMonitor.
func (hm *SequencerHealthMonitor) Stop() error {
	hm.log.Info("stopping health monitor")
	close(hm.done)
	hm.wg.Wait()

	hm.log.Info("health monitor stopped")
	return nil
}

// Subscribe implements HealthMonitor.
func (hm *SequencerHealthMonitor) Subscribe() <-chan bool {
	return hm.healthUpdateCh
}

func (hm *SequencerHealthMonitor) loop() {
	defer hm.wg.Done()

	duration := time.Duration(hm.interval) * time.Second
	ticker := time.NewTicker(duration)
	defer ticker.Stop()

	for {
		select {
		case <-hm.done:
			return
		case <-ticker.C:
			hm.healthUpdateCh <- hm.healthCheck()
		}
	}
}

// healthCheck checks the health of the sequencer by 3 criteria:
// 1. unsafe head is progressing per block time
// 2. safe head is progressing every configured batch submission interval
// 3. peer count is above the configured minimum
func (hm *SequencerHealthMonitor) healthCheck() bool {
	ctx := context.Background()
	status, err := hm.node.SyncStatus(ctx)
	if err != nil {
		hm.log.Error("health monitor failed to get sync status", "err", err)
		return false
	}

	now := uint64(time.Now().Unix())
	// allow at most one block drift for unsafe head
	if now-status.UnsafeL2.Time > hm.interval+hm.rollupCfg.BlockTime {
		hm.log.Error("unsafe head is not progressing", "lastSeenUnsafeBlock", status.UnsafeL2)
		return false
	}

	if now-status.SafeL2.Time > hm.safeInterval {
		hm.log.Error("safe head is not progressing", "safe_head_time", status.SafeL2.Time, "now", now)
		return false
	}

	stats, err := hm.p2p.PeerStats(ctx)
	if err != nil {
		hm.log.Error("health monitor failed to get peer stats", "err", err)
		return false
	}
	if uint64(stats.Connected) < hm.minPeerCount {
		hm.log.Error("peer count is below minimum", "connected", stats.Connected, "minPeerCount", hm.minPeerCount)
		return false
	}

	return true
}
