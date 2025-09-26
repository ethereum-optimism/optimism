package health

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-conductor/client"
	"github.com/ethereum-optimism/optimism/op-conductor/metrics"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/dial"
)

var (
	ErrSequencerNotHealthy         = errors.New("sequencer is not healthy")
	ErrSequencerConnectionDown     = errors.New("cannot connect to sequencer rpc endpoints")
	ErrSupervisorConnectionDown    = errors.New("cannot connect to supervisor rpc endpoint")
	ErrRollupBoostConnectionDown   = errors.New("cannot connect to rollup boost rpc endpoints")
	ErrRollupBoostPartiallyHealthy = errors.New("rollup boost is partially healthy, meaning that rbuilder is not healthy but the execution client is healthy")
	ErrRollupBoostNotHealthy       = errors.New("rollup boost is not healthy")
)

// HealthMonitor defines the interface for monitoring the health of the sequencer.
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
func NewSequencerHealthMonitor(log log.Logger, metrics metrics.Metricer, interval, unsafeInterval, safeInterval, minPeerCount uint64, safeEnabled bool, rollupCfg *rollup.Config, node dial.RollupClientInterface, p2p apis.P2PClient, supervisor SupervisorHealthAPI, rb client.RollupBoostClient, elP2pClient client.ElP2PClient, minElP2pPeers uint64, rollupBoostToleratePartialHealthinessToleranceLimit uint64, rollupBoostToleratePartialHealthinessToleranceIntervalSeconds uint64) HealthMonitor {
	hm := &SequencerHealthMonitor{
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
		supervisor:     supervisor,
		rb:             rb,
	}

	if elP2pClient != nil {
		hm.elP2p = &ElP2pHealthMonitor{
			log:          log,
			minPeerCount: minElP2pPeers,
			elP2pClient:  elP2pClient,
		}
	}
	if rollupBoostToleratePartialHealthinessToleranceLimit != 0 {
		var err error
		hm.timeTolerantRollupBoostPartialHealthinessMgr, err = NewTimeBoundedRotatingCounter(rollupBoostToleratePartialHealthinessToleranceIntervalSeconds, rollupBoostToleratePartialHealthinessToleranceLimit)
		if err != nil {
			panic(fmt.Errorf("failed to setup health monitor: %w", err))
		}
	}

	return hm
}

type ElP2pHealthMonitor struct {
	log          log.Logger
	minPeerCount uint64
	elP2pClient  client.ElP2PClient
}

// this is a type of counter which keeps on incrementing until its reset interval is hit
// this can be used to track time-based rate-limit, error counts, etc.
type timeBoundedRotatingCounter struct {
	resetIntervalSeconds uint64
	maxValue             uint64
	timeProviderFn       func() uint64

	mut           *sync.RWMutex
	temporalCache map[int64]uint64
}

func NewTimeBoundedRotatingCounter(resetIntervalSeconds, maxValue uint64) (*timeBoundedRotatingCounter, error) {
	if resetIntervalSeconds == 0 {
		panic("reset interval seconds must be more than 0")
	}
	return &timeBoundedRotatingCounter{
		resetIntervalSeconds: resetIntervalSeconds,
		maxValue:             maxValue,
		mut:                  &sync.RWMutex{},
		temporalCache:        map[int64]uint64{},
		timeProviderFn:       currentTimeProvicer,
	}, nil
}

func (t *timeBoundedRotatingCounter) Increment() (uint64, error) {
	// let's take `resetIntervalSeconds` as 60s
	// truncatedTimestamp is current timestamp rounded off by 60s (resetIntervalSeconds)
	// thereby generating a value which stays same until the next 60s helping track and incrementing the counter corresponding to it for the next 60s
	currentTsSeconds := t.timeProviderFn()
	truncatedTimestamp := int64(currentTsSeconds / t.resetIntervalSeconds)
	t.mut.Lock()
	defer t.mut.Unlock()

	// a lazy cleanup subroutine to the clean the cache when it's grown enough, preventing memory leaks
	defer func() {
		if len(t.temporalCache) > 1000 {
			newCache := map[int64]uint64{
				truncatedTimestamp: t.temporalCache[truncatedTimestamp],
			}
			t.temporalCache = newCache // garbage collector should take care of the old cache
		}
	}()

	if t.maxValue == 0 || t.temporalCache[truncatedTimestamp] < t.maxValue {
		t.temporalCache[truncatedTimestamp]++
		return t.temporalCache[truncatedTimestamp], nil
	}
	return 0, fmt.Errorf("counter at its max value, please wait %ds for it to be reset", (t.resetIntervalSeconds - (currentTsSeconds % t.resetIntervalSeconds)))
}

func (t *timeBoundedRotatingCounter) CurrentValue() uint64 {
	currentTsSeconds := time.Now().Unix()
	truncatedTimestamp := currentTsSeconds / int64(t.resetIntervalSeconds)
	t.mut.RLock()
	defer t.mut.RUnlock()
	return t.temporalCache[truncatedTimestamp]
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

	node                                         dial.RollupClientInterface
	p2p                                          apis.P2PClient
	supervisor                                   SupervisorHealthAPI
	rb                                           client.RollupBoostClient
	elP2p                                        *ElP2pHealthMonitor
	timeTolerantRollupBoostPartialHealthinessMgr *timeBoundedRotatingCounter
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
// 1. unsafe head is not too far behind now (measured by unsafeInterval)
// 2. safe head is progressing every configured batch submission interval
// 3. peer count is above the configured minimum
func (hm *SequencerHealthMonitor) healthCheck(ctx context.Context) error {
	err := hm.checkNode(ctx)
	if err != nil {
		return err
	}

	if hm.elP2p != nil {
		err = hm.elP2p.checkElP2p(ctx)
		if err != nil {
			return err
		}
	}

	err = hm.checkRollupBoost(ctx)
	if err != nil {
		return err
	}

	hm.log.Info("sequencer is healthy")
	return nil
}

func (hm *ElP2pHealthMonitor) checkElP2p(ctx context.Context) error {
	peerCount, err := hm.elP2pClient.PeerCount(ctx)
	if err != nil {
		return err
	}

	if peerCount < int(hm.minPeerCount) {
		hm.log.Error("el p2p peer count is below minimum", "peerCount", peerCount, "minPeerCount", hm.minPeerCount)
		return ErrSequencerNotHealthy
	}

	return nil
}
func (hm *SequencerHealthMonitor) checkNode(ctx context.Context) error {
	err := hm.checkNodeSyncStatus(ctx)
	if err != nil {
		return err
	}

	err = hm.checkNodePeerCount(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (hm *SequencerHealthMonitor) checkNodeSyncStatus(ctx context.Context) error {
	status, err := hm.node.SyncStatus(ctx)
	if err != nil {
		hm.log.Error("health monitor failed to get sync status", "err", err)
		return ErrSequencerConnectionDown
	}

	if hm.supervisor != nil {
		_, err := hm.supervisor.SyncStatus(ctx)
		if err != nil {
			hm.log.Error("health monitor failed to get supervisor sync status", "err", err)
			return ErrSupervisorConnectionDown
		}
	}

	now := hm.timeProviderFn()

	if status.UnsafeL2.Number > hm.lastSeenUnsafeNum {
		hm.lastSeenUnsafeNum = status.UnsafeL2.Number
		hm.lastSeenUnsafeTime = now
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

	return nil
}

func (hm *SequencerHealthMonitor) checkNodePeerCount(ctx context.Context) error {
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

func (hm *SequencerHealthMonitor) checkRollupBoost(ctx context.Context) error {
	// Skip the check if rollup boost client is not configured
	if hm.rb == nil {
		hm.log.Info("rollup boost client is not configured, skipping health check")
		return nil
	}

	status, err := hm.rb.Healthcheck(ctx)
	if err != nil {
		hm.log.Error("health monitor failed to get rollup boost status", "err", err)
		return ErrRollupBoostConnectionDown
	}

	switch status {
	case client.HealthStatusHealthy:
		return nil
	case client.HealthStatusPartial:
		if hm.timeTolerantRollupBoostPartialHealthinessMgr != nil {
			if _, err := hm.timeTolerantRollupBoostPartialHealthinessMgr.Increment(); err == nil {
				hm.log.Warn("[Tolerating Failure] Rollup boost is partial failure, builder is down but fallback execution client is up", "err", ErrRollupBoostPartiallyHealthy)
				return nil
			}
		}
		hm.log.Error("Rollup boost is partial failure, builder is down but fallback execution client is up", "err", ErrRollupBoostPartiallyHealthy)
		return ErrRollupBoostPartiallyHealthy

	case client.HealthStatusUnhealthy:
		hm.log.Error("Rollup boost total failure, both builder and fallback execution client are down", "err", ErrRollupBoostNotHealthy)
		return ErrRollupBoostNotHealthy
	default:
		hm.log.Error("Received unexpected health status from rollup boost", "status", status)
		return fmt.Errorf("unexpected rollup boost health status: %s", status)
	}
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
