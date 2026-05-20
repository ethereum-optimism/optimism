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
	ErrRollupBoostConnectionDown   = errors.New("cannot connect to rollup boost rpc endpoints")
	ErrRollupBoostPartiallyHealthy = errors.New("rollup boost is partially healthy, meaning that rbuilder is not healthy but the execution client is healthy")
	ErrRollupBoostNotHealthy       = errors.New("rollup boost is not healthy")
)

const (
	// defaultRecoveringWindowSize is the shared health-check grace window used when
	// interop reorg leniency is enabled. Unsafe-head recovery uses it to require
	// unsafe-head time to outpace wall-clock time. Sync-status RPC availability
	// and CL peer-count checks use it as a rolling window: all failed observations
	// fail the check, all successful observations restore Success, and mixed or
	// not-yet-full windows remain Inconclusive.
	defaultRecoveringWindowSize = 5
)

type rollingWindowState string

const (
	// rollingWindowSuccess means the full rolling window contains only successes.
	rollingWindowSuccess rollingWindowState = "Success"
	// rollingWindowFailed means the full rolling window contains only failures.
	rollingWindowFailed rollingWindowState = "Failed"
	// rollingWindowInconclusive means the window is not full or contains mixed results.
	rollingWindowInconclusive rollingWindowState = "Inconclusive"
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
// interopReorgLeniency enables experimental interop reorg recovery leniency.
// recoveringWindowSize is the number of observations used by lenient rolling/recovery windows.
// rollupBoostHealthChecker is an optional health checker for rollup-boost (either standard or next client).
func NewSequencerHealthMonitor(log log.Logger, metricer metrics.Metricer, interval, unsafeInterval, safeInterval, minPeerCount uint64, safeEnabled bool, interopReorgLeniency bool, recoveringWindowSize uint64, rollupCfg *rollup.Config, node dial.RollupClientInterface, p2p apis.P2PClient, rollupBoostHealthChecker client.RollupBoostHealthChecker, elP2pClient client.ElP2PClient, minElP2pPeers uint64, rollupBoostToleratePartialHealthinessToleranceLimit uint64, rollupBoostToleratePartialHealthinessToleranceIntervalSeconds uint64) HealthMonitor {
	if metricer == nil {
		metricer = metrics.NoopMetrics
	}

	hm := &SequencerHealthMonitor{
		log:                      log,
		metrics:                  metricer,
		interval:                 interval,
		healthUpdateCh:           make(chan error),
		rollupCfg:                rollupCfg,
		unsafeInterval:           unsafeInterval,
		safeEnabled:              safeEnabled,
		safeInterval:             safeInterval,
		minPeerCount:             minPeerCount,
		interopReorgLeniency:     interopReorgLeniency,
		recoveringWindowSize:     normalizeRecoveringWindowSize(recoveringWindowSize),
		timeProviderFn:           currentTimeProvider,
		node:                     node,
		p2p:                      p2p,
		rollupBoostHealthChecker: rollupBoostHealthChecker,
	}

	if elP2pClient != nil {
		hm.elP2p = &ElP2pHealthMonitor{
			log:          log,
			minPeerCount: minElP2pPeers,
			elP2pClient:  elP2pClient,
		}
	}
	if rollupBoostToleratePartialHealthinessToleranceLimit != 0 {
		hm.rollupBoostPartialHealthinessToleranceLimit = rollupBoostToleratePartialHealthinessToleranceLimit
		var err error
		hm.rollupBoostPartialHealthinessToleranceCounter, err = NewTimeBoundedRotatingCounter(rollupBoostToleratePartialHealthinessToleranceIntervalSeconds)
		if err != nil {
			panic(fmt.Errorf("failed to setup health monitor: %w", err))
		}
	}
	hm.recordStaticHealthCheckMetrics()

	return hm
}

type ElP2pHealthMonitor struct {
	log          log.Logger
	minPeerCount uint64
	elP2pClient  client.ElP2PClient
}

// SequencerHealthMonitor monitors sequencer health.
type SequencerHealthMonitor struct {
	log     log.Logger
	metrics metrics.Metricer
	cancel  context.CancelFunc
	wg      sync.WaitGroup

	rollupCfg            *rollup.Config
	unsafeInterval       uint64
	safeEnabled          bool
	safeInterval         uint64
	minPeerCount         uint64
	interopReorgLeniency bool
	recoveringWindowSize uint64
	interval             uint64
	healthUpdateCh       chan error
	lastSeenUnsafeNum    uint64
	lastSeenUnsafeTime   uint64
	syncStatusWindow     rollingWindowTracker
	peerCountWindow      rollingWindowTracker

	timeProviderFn func() uint64

	node                                          dial.RollupClientInterface
	p2p                                           apis.P2PClient
	rollupBoostHealthChecker                      client.RollupBoostHealthChecker
	elP2p                                         *ElP2pHealthMonitor
	rollupBoostPartialHealthinessToleranceLimit   uint64
	rollupBoostPartialHealthinessToleranceCounter *timeBoundedRotatingCounter

	// Recovering state. When pollsInRecovery is zero, the sequencer is not recovering.
	initialLagInRecovery        uint64
	recoveryWindowStartLag      uint64
	recoveryWindowStartWallTime uint64
	recoveryWindowStartUnsafe   uint64
	recoveryWindowStartNum      uint64
	pollsInRecovery             uint64
	pollsInRecoveryWindow       uint64
}

var _ HealthMonitor = (*SequencerHealthMonitor)(nil)

type rollingWindowTracker struct {
	observations []bool
	next         uint64
	count        uint64
	successes    uint64
}

func normalizeRecoveringWindowSize(windowSize uint64) uint64 {
	if windowSize == 0 {
		return defaultRecoveringWindowSize
	}
	return windowSize
}

func (hm *SequencerHealthMonitor) recoveringWindowSizeOrDefault() uint64 {
	return normalizeRecoveringWindowSize(hm.recoveringWindowSize)
}

func (t *rollingWindowTracker) observe(success bool, windowSize uint64) rollingWindowState {
	windowSize = normalizeRecoveringWindowSize(windowSize)
	if t.count > windowSize || uint64(len(t.observations)) > windowSize {
		t.observations = nil
		t.next = 0
		t.count = 0
		t.successes = 0
	}
	if t.count < windowSize {
		t.observations = append(t.observations, success)
		t.count++
		if success {
			t.successes++
		}
		return t.state(windowSize)
	}

	if t.observations[t.next] {
		t.successes--
	}
	t.observations[t.next] = success
	if success {
		t.successes++
	}
	t.next = (t.next + 1) % windowSize
	return t.state(windowSize)
}

func (t *rollingWindowTracker) state(windowSize uint64) rollingWindowState {
	switch {
	case t.count < windowSize:
		return rollingWindowInconclusive
	case t.successes == windowSize:
		return rollingWindowSuccess
	case t.successes == 0:
		return rollingWindowFailed
	default:
		return rollingWindowInconclusive
	}
}

func (hm *SequencerHealthMonitor) recordStaticHealthCheckMetrics() {
	if hm.metrics == nil {
		return
	}
	hm.metrics.RecordHealthCheckConfig(
		hm.interval,
		hm.unsafeInterval,
		hm.safeInterval,
		hm.minPeerCount,
		hm.recoveringWindowSizeOrDefault(),
		hm.safeEnabled,
		hm.interopReorgLeniency,
	)
	if !hm.safeEnabled {
		hm.metrics.RecordHealthCheckStatus(metrics.HealthCheckSafeLag, metrics.HealthCheckStatusDisabled)
	}
	hm.metrics.RecordUnsafeHeadRecovery(false, 0, 0, 0, 0, 0, 0, 0)
}

func (hm *SequencerHealthMonitor) recordHealthCheckHeads(statusUnsafeNum, statusUnsafeTime, statusSafeNum, statusSafeTime, unsafeLag, safeLag uint64) {
	if hm.metrics == nil {
		return
	}
	hm.metrics.RecordHealthCheckHeads(statusUnsafeNum, statusUnsafeTime, statusSafeNum, statusSafeTime, unsafeLag, safeLag)
}

func (hm *SequencerHealthMonitor) recordPeerCount(peerCount uint64) {
	if hm.metrics == nil {
		return
	}
	hm.metrics.RecordHealthCheckPeerCount(peerCount, hm.minPeerCount)
}

func (hm *SequencerHealthMonitor) recordRollingWindow(check metrics.HealthCheck, tracker rollingWindowTracker, state rollingWindowState) {
	if hm.metrics == nil {
		return
	}
	successes := tracker.successes
	failures := tracker.count - successes
	hm.metrics.RecordHealthCheckWindow(check, metricWindowState(state), successes, failures, hm.recoveringWindowSizeOrDefault())
}

func (hm *SequencerHealthMonitor) recordCheckStatus(check metrics.HealthCheck, status metrics.HealthCheckStatus) {
	if hm.metrics == nil {
		return
	}
	hm.metrics.RecordHealthCheckStatus(check, status)
}

func (hm *SequencerHealthMonitor) recordCheckFailure(check metrics.HealthCheck, reason metrics.HealthCheckFailureReason) {
	if hm.metrics == nil {
		return
	}
	hm.metrics.RecordHealthCheckFailure(check, reason)
}

func (hm *SequencerHealthMonitor) recordUnsafeHeadRecovery(active bool, currentLag, wallElapsed, unsafeElapsed uint64) {
	if hm.metrics == nil {
		return
	}
	hm.metrics.RecordUnsafeHeadRecovery(
		active,
		currentLag,
		hm.initialLagInRecovery,
		hm.recoveryWindowStartLag,
		wallElapsed,
		unsafeElapsed,
		hm.pollsInRecovery,
		hm.pollsInRecoveryWindow,
	)
}

func (hm *SequencerHealthMonitor) recordUnsafeHeadRecoveryEvent(event metrics.HealthCheckRecoveryEvent) {
	if hm.metrics == nil {
		return
	}
	hm.metrics.RecordUnsafeHeadRecoveryEvent(event)
}

func metricWindowState(state rollingWindowState) metrics.HealthCheckWindowState {
	switch state {
	case rollingWindowSuccess:
		return metrics.HealthCheckWindowStateSuccess
	case rollingWindowFailed:
		return metrics.HealthCheckWindowStateFailed
	default:
		return metrics.HealthCheckWindowStateInconclusive
	}
}

// Start implements HealthMonitor.
func (hm *SequencerHealthMonitor) Start(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	hm.cancel = cancel

	hm.log.Info("starting health monitor")
	hm.recordStaticHealthCheckMetrics()
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
// 1. unsafe head is not too far behind now. When interop reorg leniency is
// enabled, a lagging unsafe head may stay healthy while recovering faster than
// wall clock.
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
	if hm.interopReorgLeniency {
		return hm.checkNodeSyncStatusInteropReorgLenient(ctx)
	}
	return hm.checkNodeSyncStatusStrict(ctx)
}

func (hm *SequencerHealthMonitor) checkNodeSyncStatusStrict(ctx context.Context) error {
	status, err := hm.node.SyncStatus(ctx)
	if err != nil {
		hm.log.Error("health monitor failed to get sync status", "err", err)
		hm.recordCheckStatus(metrics.HealthCheckSyncStatusRPC, metrics.HealthCheckStatusUnhealthy)
		hm.recordCheckFailure(metrics.HealthCheckSyncStatusRPC, metrics.HealthCheckFailureReasonRPCError)
		return ErrSequencerConnectionDown
	}
	hm.recordCheckStatus(metrics.HealthCheckSyncStatusRPC, metrics.HealthCheckStatusHealthy)

	now := hm.timeProviderFn()

	if status.UnsafeL2.Number > hm.lastSeenUnsafeNum {
		hm.lastSeenUnsafeNum = status.UnsafeL2.Number
		hm.lastSeenUnsafeTime = now
	}

	curUnsafeTimeDiff := calculateTimeDiff(now, status.UnsafeL2.Time)
	curSafeTimeDiff := calculateTimeDiff(now, status.SafeL2.Time)
	hm.recordHealthCheckHeads(status.UnsafeL2.Number, status.UnsafeL2.Time, status.SafeL2.Number, status.SafeL2.Time, curUnsafeTimeDiff, curSafeTimeDiff)
	hm.recordUnsafeHeadRecovery(false, curUnsafeTimeDiff, 0, 0)
	if curUnsafeTimeDiff > hm.unsafeInterval {
		hm.log.Error(
			"unsafe head is falling behind the unsafe interval",
			"now", now,
			"unsafe_head_num", status.UnsafeL2.Number,
			"unsafe_head_time", status.UnsafeL2.Time,
			"unsafe_interval", hm.unsafeInterval,
			"cur_unsafe_time_diff", curUnsafeTimeDiff,
		)
		hm.recordCheckStatus(metrics.HealthCheckUnsafeLag, metrics.HealthCheckStatusUnhealthy)
		hm.recordCheckFailure(metrics.HealthCheckUnsafeLag, metrics.HealthCheckFailureReasonLagExceeded)
		return ErrSequencerNotHealthy
	}
	hm.recordCheckStatus(metrics.HealthCheckUnsafeLag, metrics.HealthCheckStatusHealthy)

	if !hm.safeEnabled {
		hm.recordCheckStatus(metrics.HealthCheckSafeLag, metrics.HealthCheckStatusDisabled)
		return nil
	}

	if curSafeTimeDiff > hm.safeInterval {
		hm.log.Error(
			"safe head is not progressing as expected",
			"now", now,
			"safe_head_num", status.SafeL2.Number,
			"safe_head_time", status.SafeL2.Time,
			"safe_interval", hm.safeInterval,
		)
		hm.recordCheckStatus(metrics.HealthCheckSafeLag, metrics.HealthCheckStatusUnhealthy)
		hm.recordCheckFailure(metrics.HealthCheckSafeLag, metrics.HealthCheckFailureReasonLagExceeded)
		return ErrSequencerNotHealthy
	}
	hm.recordCheckStatus(metrics.HealthCheckSafeLag, metrics.HealthCheckStatusHealthy)

	return nil
}

func (hm *SequencerHealthMonitor) checkNodeSyncStatusInteropReorgLenient(ctx context.Context) error {
	windowSize := hm.recoveringWindowSizeOrDefault()
	status, err := hm.node.SyncStatus(ctx)
	if err != nil {
		state := hm.syncStatusWindow.observe(false, windowSize)
		hm.recordRollingWindow(metrics.HealthCheckSyncStatusRPC, hm.syncStatusWindow, state)
		if state == rollingWindowFailed {
			hm.log.Error(
				"health monitor failed to get sync status",
				"err", err,
				"window_state", state,
				"window_size", windowSize,
			)
			hm.recordCheckStatus(metrics.HealthCheckSyncStatusRPC, metrics.HealthCheckStatusUnhealthy)
			hm.recordCheckFailure(metrics.HealthCheckSyncStatusRPC, metrics.HealthCheckFailureReasonRPCError)
			return ErrSequencerConnectionDown
		}
		hm.log.Warn(
			"health monitor temporarily failed to get sync status",
			"err", err,
			"window_state", state,
			"window_size", windowSize,
		)
		hm.recordCheckStatus(metrics.HealthCheckSyncStatusRPC, metrics.HealthCheckStatusWarning)
		return nil
	}
	state := hm.syncStatusWindow.observe(true, windowSize)
	hm.recordRollingWindow(metrics.HealthCheckSyncStatusRPC, hm.syncStatusWindow, state)
	hm.recordCheckStatus(metrics.HealthCheckSyncStatusRPC, metrics.HealthCheckStatusHealthy)

	now := hm.timeProviderFn()

	if status.UnsafeL2.Number > hm.lastSeenUnsafeNum {
		hm.lastSeenUnsafeNum = status.UnsafeL2.Number
		hm.lastSeenUnsafeTime = now
	}

	curUnsafeLag := calculateTimeDiff(now, status.UnsafeL2.Time)
	curSafeLag := calculateTimeDiff(now, status.SafeL2.Time)
	hm.recordHealthCheckHeads(status.UnsafeL2.Number, status.UnsafeL2.Time, status.SafeL2.Number, status.SafeL2.Time, curUnsafeLag, curSafeLag)
	switch {
	case curUnsafeLag <= hm.unsafeInterval:
		if hm.pollsInRecovery > 0 {
			hm.log.Info(
				"sequencer recovered from unsafe-head lag",
				"polls_in_recovery", hm.pollsInRecovery,
				"initial_lag_in_recovery", hm.initialLagInRecovery,
				"cur_unsafe_lag", curUnsafeLag,
			)
			hm.recordUnsafeHeadRecoveryEvent(metrics.HealthCheckRecoveryRecovered)
			hm.clearUnsafeHeadRecovery()
		}
		hm.recordCheckStatus(metrics.HealthCheckUnsafeLag, metrics.HealthCheckStatusHealthy)
		hm.recordUnsafeHeadRecovery(false, curUnsafeLag, 0, 0)
	case hm.pollsInRecovery == 0:
		hm.initialLagInRecovery = curUnsafeLag
		hm.pollsInRecovery = 1
		hm.resetUnsafeHeadRecoveryWindow(now, status.UnsafeL2.Time, status.UnsafeL2.Number, curUnsafeLag)
		hm.recordUnsafeHeadRecoveryEvent(metrics.HealthCheckRecoveryEntered)
		hm.recordCheckStatus(metrics.HealthCheckUnsafeLag, metrics.HealthCheckStatusRecovering)
		hm.recordUnsafeHeadRecovery(true, curUnsafeLag, 0, 0)
		hm.log.Info(
			"sequencer entering unsafe-head recovery",
			"cur_unsafe_lag", curUnsafeLag,
			"unsafe_interval", hm.unsafeInterval,
			"unsafe_head_num", status.UnsafeL2.Number,
		)
	default:
		hm.pollsInRecovery++
		hm.pollsInRecoveryWindow++
		wallElapsed := calculateTimeDiff(now, hm.recoveryWindowStartWallTime)
		unsafeElapsed := calculateTimeDiff(status.UnsafeL2.Time, hm.recoveryWindowStartUnsafe)
		if unsafeElapsed > wallElapsed {
			hm.log.Info(
				"unsafe-head outpacing wall clock during recovery",
				"cur_unsafe_lag", curUnsafeLag,
				"previous_window_start_lag", hm.recoveryWindowStartLag,
				"wall_elapsed", wallElapsed,
				"unsafe_elapsed", unsafeElapsed,
				"polls_in_recovery", hm.pollsInRecovery,
				"polls_in_recovery_window", hm.pollsInRecoveryWindow,
			)
			hm.resetUnsafeHeadRecoveryWindow(now, status.UnsafeL2.Time, status.UnsafeL2.Number, curUnsafeLag)
			hm.recordUnsafeHeadRecoveryEvent(metrics.HealthCheckRecoveryWindowReset)
			hm.recordCheckStatus(metrics.HealthCheckUnsafeLag, metrics.HealthCheckStatusRecovering)
			hm.recordUnsafeHeadRecovery(true, curUnsafeLag, 0, 0)
			break
		}
		if hm.pollsInRecoveryWindow >= windowSize {
			hm.log.Error(
				"unsafe-head recovery not outpacing wall clock",
				"cur_unsafe_lag", curUnsafeLag,
				"recovery_window_start_lag", hm.recoveryWindowStartLag,
				"recovery_window_start_num", hm.recoveryWindowStartNum,
				"wall_elapsed", wallElapsed,
				"unsafe_elapsed", unsafeElapsed,
				"polls_in_recovery", hm.pollsInRecovery,
				"polls_in_recovery_window", hm.pollsInRecoveryWindow,
			)
			hm.recordCheckStatus(metrics.HealthCheckUnsafeLag, metrics.HealthCheckStatusUnhealthy)
			hm.recordCheckFailure(metrics.HealthCheckUnsafeLag, metrics.HealthCheckFailureReasonRecoveryWindowStalled)
			hm.recordUnsafeHeadRecoveryEvent(metrics.HealthCheckRecoveryFailed)
			hm.recordUnsafeHeadRecovery(true, curUnsafeLag, wallElapsed, unsafeElapsed)
			return ErrSequencerNotHealthy
		}
		hm.recordCheckStatus(metrics.HealthCheckUnsafeLag, metrics.HealthCheckStatusRecovering)
		hm.recordUnsafeHeadRecovery(true, curUnsafeLag, wallElapsed, unsafeElapsed)
	}

	if !hm.safeEnabled {
		hm.recordCheckStatus(metrics.HealthCheckSafeLag, metrics.HealthCheckStatusDisabled)
		return nil
	}

	if curSafeLag > hm.safeInterval {
		hm.log.Error(
			"safe head is not progressing as expected",
			"now", now,
			"safe_head_num", status.SafeL2.Number,
			"safe_head_time", status.SafeL2.Time,
			"safe_interval", hm.safeInterval,
		)
		hm.recordCheckStatus(metrics.HealthCheckSafeLag, metrics.HealthCheckStatusUnhealthy)
		hm.recordCheckFailure(metrics.HealthCheckSafeLag, metrics.HealthCheckFailureReasonLagExceeded)
		return ErrSequencerNotHealthy
	}
	hm.recordCheckStatus(metrics.HealthCheckSafeLag, metrics.HealthCheckStatusHealthy)

	return nil
}

func (hm *SequencerHealthMonitor) resetUnsafeHeadRecoveryWindow(now, unsafeTime, unsafeNum, curUnsafeLag uint64) {
	hm.recoveryWindowStartLag = curUnsafeLag
	hm.recoveryWindowStartWallTime = now
	hm.recoveryWindowStartUnsafe = unsafeTime
	hm.recoveryWindowStartNum = unsafeNum
	hm.pollsInRecoveryWindow = 1
}

func (hm *SequencerHealthMonitor) clearUnsafeHeadRecovery() {
	hm.initialLagInRecovery = 0
	hm.recoveryWindowStartLag = 0
	hm.recoveryWindowStartWallTime = 0
	hm.recoveryWindowStartUnsafe = 0
	hm.recoveryWindowStartNum = 0
	hm.pollsInRecovery = 0
	hm.pollsInRecoveryWindow = 0
}

func (hm *SequencerHealthMonitor) checkNodePeerCount(ctx context.Context) error {
	if hm.interopReorgLeniency {
		return hm.checkNodePeerCountInteropReorgLenient(ctx)
	}
	return hm.checkNodePeerCountStrict(ctx)
}

func (hm *SequencerHealthMonitor) checkNodePeerCountStrict(ctx context.Context) error {
	stats, err := hm.p2p.PeerStats(ctx)
	if err != nil {
		hm.log.Error("health monitor failed to get peer stats", "err", err)
		hm.recordCheckStatus(metrics.HealthCheckPeerCount, metrics.HealthCheckStatusUnhealthy)
		hm.recordCheckFailure(metrics.HealthCheckPeerCount, metrics.HealthCheckFailureReasonRPCError)
		return ErrSequencerConnectionDown
	}
	if uint64(stats.Connected) < hm.minPeerCount {
		hm.recordPeerCount(uint64(stats.Connected))
		hm.log.Error("peer count is below minimum", "connected", stats.Connected, "minPeerCount", hm.minPeerCount)
		hm.recordCheckStatus(metrics.HealthCheckPeerCount, metrics.HealthCheckStatusUnhealthy)
		hm.recordCheckFailure(metrics.HealthCheckPeerCount, metrics.HealthCheckFailureReasonPeerCountBelowMin)
		return ErrSequencerNotHealthy
	}
	hm.recordPeerCount(uint64(stats.Connected))
	hm.recordCheckStatus(metrics.HealthCheckPeerCount, metrics.HealthCheckStatusHealthy)

	return nil
}

func (hm *SequencerHealthMonitor) checkNodePeerCountInteropReorgLenient(ctx context.Context) error {
	windowSize := hm.recoveringWindowSizeOrDefault()
	stats, err := hm.p2p.PeerStats(ctx)
	if err != nil {
		state := hm.peerCountWindow.observe(false, windowSize)
		hm.recordRollingWindow(metrics.HealthCheckPeerCount, hm.peerCountWindow, state)
		if state == rollingWindowFailed {
			hm.log.Error("health monitor failed to get peer stats", "err", err, "window_state", state, "window_size", windowSize)
			hm.recordCheckStatus(metrics.HealthCheckPeerCount, metrics.HealthCheckStatusUnhealthy)
			hm.recordCheckFailure(metrics.HealthCheckPeerCount, metrics.HealthCheckFailureReasonRPCError)
			return ErrSequencerConnectionDown
		}
		hm.log.Warn("health monitor temporarily failed to get peer stats", "err", err, "window_state", state, "window_size", windowSize)
		hm.recordCheckStatus(metrics.HealthCheckPeerCount, metrics.HealthCheckStatusWarning)
		return nil
	}
	hm.recordPeerCount(uint64(stats.Connected))
	if uint64(stats.Connected) < hm.minPeerCount {
		state := hm.peerCountWindow.observe(false, windowSize)
		hm.recordRollingWindow(metrics.HealthCheckPeerCount, hm.peerCountWindow, state)
		if state == rollingWindowFailed {
			hm.log.Error("peer count is below minimum", "connected", stats.Connected, "minPeerCount", hm.minPeerCount, "window_state", state, "window_size", windowSize)
			hm.recordCheckStatus(metrics.HealthCheckPeerCount, metrics.HealthCheckStatusUnhealthy)
			hm.recordCheckFailure(metrics.HealthCheckPeerCount, metrics.HealthCheckFailureReasonPeerCountBelowMin)
			return ErrSequencerNotHealthy
		}
		hm.log.Warn("peer count is temporarily below minimum", "connected", stats.Connected, "minPeerCount", hm.minPeerCount, "window_state", state, "window_size", windowSize)
		hm.recordCheckStatus(metrics.HealthCheckPeerCount, metrics.HealthCheckStatusWarning)
		return nil
	}
	state := hm.peerCountWindow.observe(true, windowSize)
	hm.recordRollingWindow(metrics.HealthCheckPeerCount, hm.peerCountWindow, state)
	hm.recordCheckStatus(metrics.HealthCheckPeerCount, metrics.HealthCheckStatusHealthy)

	return nil
}

func (hm *SequencerHealthMonitor) checkRollupBoost(ctx context.Context) error {
	// Skip the check if rollup boost health checker is not configured
	if hm.rollupBoostHealthChecker == nil {
		hm.log.Debug("rollup-boost health checker is not configured, skipping health check")
		return nil
	}

	status, err := hm.rollupBoostHealthChecker.Healthcheck(ctx)
	if err != nil {
		hm.log.Error("health monitor failed to get rollup-boost status", "err", err)
		return ErrRollupBoostConnectionDown
	}

	return hm.handleRollupBoostStatus(status)
}

func (hm *SequencerHealthMonitor) handleRollupBoostStatus(status client.HealthStatus) error {
	switch status {
	case client.HealthStatusHealthy:
		return nil
	case client.HealthStatusPartial:
		if hm.rollupBoostPartialHealthinessToleranceCounter != nil && hm.rollupBoostPartialHealthinessToleranceCounter.CurrentValue() < hm.rollupBoostPartialHealthinessToleranceLimit {
			latestValue := hm.rollupBoostPartialHealthinessToleranceCounter.Increment()
			hm.log.Debug("Rollup boost partial unhealthiness failure tolerated", "currentValue", latestValue, "limit", hm.rollupBoostPartialHealthinessToleranceLimit)
			return nil
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

func currentTimeProvider() uint64 {
	return uint64(time.Now().Unix())
}
