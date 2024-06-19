package health

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-conductor/metrics"
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
	Start(ctx context.Context) error
	// Stop stops the health check.
	Stop() error
}

// NewSequencerHealthMonitor creates a new sequencer health monitor.
// interval is the interval between health checks measured in seconds.
// safeInterval is the interval between safe head progress measured in seconds.
// minPeerCount is the minimum number of peers required for the sequencer to be healthy.
func NewSequencerHealthMonitor(log log.Logger, metrics metrics.Metricer, interval, unsafeInterval, safeInterval, minPeerCount uint64, safeEnabled bool, rollupCfg *rollup.Config, node dial.RollupClientInterface, p2p p2p.API) HealthMonitor {
	return &SequencerHealthMonitor{
		log:            log,
		metrics:        metrics,
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
	log     log.Logger
	metrics metrics.Metricer
	cancel  context.CancelFunc
	wg      sync.WaitGroup

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
func (hm *SequencerHealthMonitor) Start(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	hm.cancel = cancel

	hm.log.Info("starting health monitor")
	hm.wg.Add(1)
	go hm.loop(ctx)

	hm.log.Info("health monitor started")
	return nil
}

// Stop implements HealthMonitor.
func (hm *SequencerHealthMonitor) Stop() error {
	hm.log.Info("stopping health monitor")
	hm.cancel()
	hm.wg.Wait()

	hm.log.Info("health monitor stopped")
	return nil
}

// Subscribe implements HealthMonitor.
func (hm *SequencerHealthMonitor) Subscribe() <-chan error {
	return hm.healthUpdateCh
}

func (hm *SequencerHealthMonitor) loop(ctx context.Context) {
	defer hm.wg.Done()

	duration := time.Duration(hm.interval) * time.Second
	ticker := time.NewTicker(duration)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			err := hm.healthCheck(ctx)
			hm.metrics.RecordHealthCheck(err == nil, err)
			// Ensure that we exit cleanly if told to shutdown while still waiting to publish the health update
			select {
			case hm.healthUpdateCh <- err:
				continue
			case <-ctx.Done():
				return
			}
		}
	}
}

// healthCheck checks the health of the sequencer by 3 criteria:
// 1. unsafe head is progressing per block time
// 2. unsafe head is not too far behind now (measured by unsafeInterval)
// 3. safe head is progressing every configured batch submission interval
// 4. peer count is above the configured minimum
func (hm *SequencerHealthMonitor) healthCheck(ctx context.Context) error {
	status, err := hm.node.SyncStatus(ctx)
	if err != nil {
		hm.log.Error("health monitor failed to get sync status", "err", err)
		return ErrSequencerConnectionDown
	}

	now := hm.timeProviderFn()

	var timeDiff, blockDiff, expectedBlocks uint64
	if hm.lastSeenUnsafeNum != 0 {
		timeDiff = calculateTimeDiff(now, hm.lastSeenUnsafeTime)
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
			"time_diff", timeDiff,
			"block_diff", blockDiff,
			"expected_blocks", expectedBlocks,
		)
		return ErrSequencerNotHealthy
	}

	curUnsafeTimeDiff := calculateTimeDiff(now, status.UnsafeL2.Time)
	if curUnsafeTimeDiff > hm.unsafeInterval {
		hm.log.Error(
			"unsafe head is falling behind the unsafe interval",
			"now", now,
			"unsafe_head_num", status.UnsafeL2.Number,
			"unsafe_head_time", status.UnsafeL2.Time,
			"unsafe_interval", hm.unsafeInterval,
			"cur_unsafe_time_diff", curUnsafeTimeDiff,
		)
		return ErrSequencerNotHealthy
	}

	if hm.safeEnabled && calculateTimeDiff(now, status.SafeL2.Time) > hm.safeInterval {
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

	hm.log.Info("sequencer is healthy")
	return nil
}

func calculateTimeDiff(now, then uint64) uint64 {
	if now < then {
		return 0
	}
	return now - then
}

func currentTimeProvicer() uint64 {
	return uint64(time.Now().Unix())
}
