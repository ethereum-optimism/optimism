package health

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/p2p"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/dial"
)

var (
	ErrSequencerNotHealthy     = errors.New("sequencer is not healthy")
	ErrSequencerConnectionDown = errors.New("cannot connect to sequencer rpc endpoints")
)

// HealthMonitor defines the interface for monitoring the health of the sequencer.
//
//go:generate mockery --name HealthMonitor --output mocks/ --with-expecter=true
type HealthMonitor interface {
	// Subscribe returns a channel that will be notified for every health check.
	Subscribe() <-chan error
	// Start starts the health check.
	Start() error
	// Stop stops the health check.
	Stop() error
}

// NewSequencerHealthMonitor creates a new sequencer health monitor.
// interval is the interval between health checks measured in seconds.
// safeInterval is the interval between safe head progress measured in seconds.
// minPeerCount is the minimum number of peers required for the sequencer to be healthy.
func NewSequencerHealthMonitor(log log.Logger, interval, unsafeInterval, safeInterval, minPeerCount uint64, safeEnabled bool, rollupCfg *rollup.Config, node dial.RollupClientInterface, p2p p2p.API) HealthMonitor {
	return &SequencerHealthMonitor{
		log:            log,
		done:           make(chan struct{}),
		interval:       interval,
		healthUpdateCh: make(chan error),
		rollupCfg:      rollupCfg,
		unsafeInterval: unsafeInterval,
		safeEnabled:    safeEnabled,
		safeInterval:   safeInterval,
		minPeerCount:   minPeerCount,
		timeProviderFn: currentTimeProvicer,
		node:           node,
		p2p:            p2p,
	}
}

// SequencerHealthMonitor monitors sequencer health.
type SequencerHealthMonitor struct {
	log  log.Logger
	done chan struct{}
	wg   sync.WaitGroup

	rollupCfg          *rollup.Config
	unsafeInterval     uint64
	safeEnabled        bool
	safeInterval       uint64
	minPeerCount       uint64
	interval           uint64
	healthUpdateCh     chan error
	lastSeenUnsafeNum  uint64
	lastSeenUnsafeTime uint64

	timeProviderFn func() uint64

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
func (hm *SequencerHealthMonitor) Subscribe() <-chan error {
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
// 2. unsafe head is not too far behind now (measured by unsafeInterval)
// 3. safe head is progressing every configured batch submission interval
// 4. peer count is above the configured minimum
func (hm *SequencerHealthMonitor) healthCheck() error {
	ctx := context.Background()
	status, err := hm.node.SyncStatus(ctx)
	if err != nil {
		hm.log.Error("health monitor failed to get sync status", "err", err)
		return ErrSequencerConnectionDown
	}

	now := hm.timeProviderFn()

	var timeDiff, blockDiff, expectedBlocks uint64
	if hm.lastSeenUnsafeNum != 0 {
		timeDiff = now - hm.lastSeenUnsafeTime
		blockDiff = status.UnsafeL2.Number - hm.lastSeenUnsafeNum
		// how many blocks do we expect to see, minus 1 to account for edge case with respect to time.
		// for example, if diff = 2.001s and block time = 2s, expecting to see 1 block could potentially cause sequencer to be considered unhealthy.
		expectedBlocks = timeDiff / hm.rollupCfg.BlockTime
		if expectedBlocks > 0 {
			expectedBlocks--
		}
	}
	if status.UnsafeL2.Number > hm.lastSeenUnsafeNum {
		hm.lastSeenUnsafeNum = status.UnsafeL2.Number
		hm.lastSeenUnsafeTime = now
	}

	if timeDiff > hm.rollupCfg.BlockTime && expectedBlocks > blockDiff {
		hm.log.Error(
			"unsafe head is not progressing as expected",
			"now", now,
			"unsafe_head_num", status.UnsafeL2.Number,
			"last_seen_unsafe_num", hm.lastSeenUnsafeNum,
			"last_seen_unsafe_time", hm.lastSeenUnsafeTime,
			"unsafe_interval", hm.unsafeInterval,
		)
		return ErrSequencerNotHealthy
	}

	if now-status.UnsafeL2.Time > hm.unsafeInterval {
		hm.log.Error(
			"unsafe head is not progressing as expected",
			"now", now,
			"unsafe_head_num", status.UnsafeL2.Number,
			"unsafe_head_time", status.UnsafeL2.Time,
			"unsafe_interval", hm.unsafeInterval,
		)
		return ErrSequencerNotHealthy
	}

	if hm.safeEnabled && now-status.SafeL2.Time > hm.safeInterval {
		hm.log.Error(
			"safe head is not progressing as expected",
			"now", now,
			"safe_head_num", status.SafeL2.Number,
			"safe_head_time", status.SafeL2.Time,
			"safe_interval", hm.safeInterval,
		)
		return ErrSequencerNotHealthy
	}

	stats, err := hm.p2p.PeerStats(ctx)
	if err != nil {
		hm.log.Error("health monitor failed to get peer stats", "err", err)
		return ErrSequencerConnectionDown
	}
	if uint64(stats.Connected) < hm.minPeerCount {
		hm.log.Error("peer count is below minimum", "connected", stats.Connected, "minPeerCount", hm.minPeerCount)
		return ErrSequencerNotHealthy
	}

	return nil
}

func currentTimeProvicer() uint64 {
	return uint64(time.Now().Unix())
}
