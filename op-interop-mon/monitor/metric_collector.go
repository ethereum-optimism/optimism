package monitor

import (
	"context"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/log"
)

type InteropMessageMetrics interface {
	RecordMessageStatus(executingChainID string, initiatingChainID string, status string, count float64)
	RecordTerminalStatusChange(executingChainID string, initiatingChainID string, count float64)
	RecordExecutingBlockRange(chainID string, min uint64, max uint64)
	RecordInitiatingBlockRange(chainID string, min uint64, max uint64)
	RecordInitiatingReorg(executingChainID string, initiatingChainID string)
	RecordFilterDivergence(executingChainID string, initiatingChainID string, monitorStatus string, filterStatus string)
	RecordFilterFailsafe(enabled bool)
	RecordSupernodeUp(endpoint string, up bool)
	RecordSupernodeSafeHead(chainID string, level string, blockNumber uint64)
	RecordCrossSafetyViolation(executingChainID string, initiatingChainID string)
}

type MetricCollector struct {
	updaters map[eth.ChainID]Updater

	closed chan struct{}
	log    log.Logger
	m      InteropMessageMetrics

	// ctx bounds in-flight observer RPCs; cancel is invoked on Stop so shutdown
	// does not wait on outstanding filter/supernode calls.
	ctx    context.Context
	cancel context.CancelFunc

	// filterObserver, when set, cross-checks the monitor's verdict against the
	// op-interop-filter (read-only). nil disables the filter observer.
	filterObserver *FilterObserver

	// supernodeObservers, when non-empty, observe each op-supernode (read-only):
	// liveness, per-chain heads, and cross-safety violations.
	supernodeObservers []*SupernodeObserver
}

func NewMetricCollector(log log.Logger, m InteropMessageMetrics, updaters map[eth.ChainID]Updater) *MetricCollector {
	return &MetricCollector{
		log:      log,
		m:        m,
		updaters: updaters,
		closed:   make(chan struct{}),
		// Default so CollectMetrics is safe if invoked before Start (e.g. in tests);
		// Start replaces this with a cancelable child of the service context.
		ctx: context.Background(),
	}
}

func (m *MetricCollector) Start(ctx context.Context) error {
	m.log.Info("Starting metric collector")
	m.ctx, m.cancel = context.WithCancel(ctx)
	go m.Run()
	return nil
}

func (m *MetricCollector) Stopped() bool {
	select {
	case <-m.closed:
		return true
	default:
		return false
	}
}

// Run is the main loop for the metric collector
func (m *MetricCollector) Run() {
	// set up a ticker to run every 1s
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-m.closed:
			return
		case <-ticker.C:
			m.CollectMetrics()
		}
	}
}

func (m *MetricCollector) Stop() error {
	close(m.closed)
	if m.cancel != nil {
		m.cancel()
	}
	return nil
}

// CollectMetrics scans the jobMaps, consolidates them, and updates the metrics
func (m *MetricCollector) CollectMetrics() {
	chains := []eth.ChainID{}
	jobMap := map[JobID]*Job{}
	for chainID, updater := range m.updaters {
		chains = append(chains, chainID)
		jobMap = updater.CollectForMetrics(jobMap)
	}

	// Initialize all metrics with zero values
	// Message Status: [executingChainID][initiatingChainID][status]
	// Terminal Status Changes: [executingChainID][initiatingChainID]
	// Executing Block Range: [chainID][min, max]
	// Initiating Block Range: [chainID][min, max]
	messageStatus := map[eth.ChainID]map[eth.ChainID]map[string]int{}
	terminalStatusChanges := map[eth.ChainID]map[eth.ChainID]int{}
	executingRanges := map[eth.ChainID]struct{ min, max uint64 }{}
	initiatingRanges := map[eth.ChainID]struct{ min, max uint64 }{}
	for _, exeChain := range chains {
		executingRanges[exeChain] = struct {
			min, max uint64
		}{min: 0, max: 0}
		initiatingRanges[exeChain] = struct {
			min, max uint64
		}{min: 0, max: 0}
		messageStatus[exeChain] = map[eth.ChainID]map[string]int{}
		terminalStatusChanges[exeChain] = map[eth.ChainID]int{}
		for _, initChain := range chains {
			terminalStatusChanges[exeChain][initChain] = 0
			messageStatus[exeChain][initChain] = map[string]int{}
			for _, status := range []string{
				jobStatusValid.String(),
				jobStatusInvalid.String(),
				jobStatusExpired.String(),
				jobStatusTimestampMismatch.String(),
				jobStatusUnknown.String(),
			} {
				messageStatus[exeChain][initChain][status] = 0
			}
		}
	}

	// Process jobs and update metrics
	for _, job := range jobMap {
		// Update executing ranges
		execRange := executingRanges[job.executingChain]
		if execRange.min == 0 {
			execRange.min = job.executingBlock.Number
		}
		if job.executingBlock.Number < execRange.min {
			execRange.min = job.executingBlock.Number
		}
		if job.executingBlock.Number > execRange.max {
			execRange.max = job.executingBlock.Number
		}
		executingRanges[job.executingChain] = execRange

		// Update initiating ranges
		initRange := initiatingRanges[job.initiating.ChainID]
		if initRange.min == 0 {
			initRange.min = job.initiating.BlockNumber
		}
		if job.initiating.BlockNumber < initRange.min {
			initRange.min = job.initiating.BlockNumber
		}
		if job.initiating.BlockNumber > initRange.max {
			initRange.max = job.initiating.BlockNumber
		}
		initiatingRanges[job.initiating.ChainID] = initRange

		// Check for multiple initiating hashes
		initiatingHashes := job.InitiatingHashes()
		if len(initiatingHashes) > 1 {
			m.log.Warn("Initiating BlockNumber found multiple Blocks (reorg of initiating block)",
				"executing_chain_id", job.executingChain,
				"initiating_chain_id", job.initiating.ChainID,
				"executing_block_height", job.executingBlock.Number,
				"initiating_block_height", job.initiating.BlockNumber,
				"executing_block_hash", job.executingBlock.Hash,
				"initiating_hashes", initiatingHashes,
			)
			// Count each reorged job once, not on every collection cycle.
			if job.CountReorgOnce() {
				m.m.RecordInitiatingReorg(job.executingChain.String(), job.initiating.ChainID.String())
			}
		}

		// Collect the statuses of the job
		statuses := job.Statuses()
		if len(statuses) == 0 {
			m.log.Warn("Job has no statuses", "job", job)
			continue
		}
		current := statuses[len(statuses)-1]

		// Log invalid statuses (observe-only: no actuation)
		if current == jobStatusInvalid {
			m.log.Warn("Invalid Executing Message Detected",
				"executing_chain_id", job.executingChain,
				"initiating_chain_id", job.initiating.ChainID,
				"executing_block_height", job.executingBlock.Number,
				"initiating_block_height", job.initiating.BlockNumber,
				"executing_block_hash", job.executingBlock.Hash,
			)
		}

		// Increment the message status metrics
		messageStatus[job.executingChain][job.initiating.ChainID][current.String()]++

		// Evaluate the job for a terminal state change
		hasBeenValid := false
		hasBeenInvalid := false
		for _, state := range statuses {
			switch state {
			case jobStatusValid:
				hasBeenValid = true
			case jobStatusInvalid:
				hasBeenInvalid = true
			}
		}
		if hasBeenValid && hasBeenInvalid {
			m.log.Warn("Executing Message has been both Valid and Invalid",
				"executing_chain_id", job.executingChain,
				"initiating_chain_id", job.initiating.ChainID,
				"executing_block_height", job.executingBlock.Number,
				"initiating_block_height", job.initiating.BlockNumber,
				"executing_block_hash", job.executingBlock.Hash,
			)
			terminalStatusChanges[job.executingChain][job.initiating.ChainID]++
		}
	}

	// Update metrics for all combinations
	for executingChainID, initiatingChainMap := range messageStatus {
		for initiatingChainID, statusMap := range initiatingChainMap {
			for status, count := range statusMap {
				if status == jobStatusInvalid.String() && count > 0 {
					// invalid messages are logged as warnings, but only when any exist
					m.log.Warn("Invalid Executing Messages Detected",
						"executing_chain_id", executingChainID,
						"initiating_chain_id", initiatingChainID,
						"count", count,
					)
				} else {
					// valid or unknown messages are logged as debug
					m.log.Debug("Updating Executing Message Status Count",
						"executing_chain_id", executingChainID,
						"initiating_chain_id", initiatingChainID,
						"status", status,
						"count", count,
					)
				}
				m.m.RecordMessageStatus(
					executingChainID.String(),
					initiatingChainID.String(),
					status,
					float64(count),
				)
			}
		}
	}

	// Record terminal status changes for all combinations
	for executingChainID, initiatingChainIDMap := range terminalStatusChanges {
		for initiatingChainID, count := range initiatingChainIDMap {
			m.m.RecordTerminalStatusChange(
				executingChainID.String(),
				initiatingChainID.String(),
				float64(count),
			)
		}
	}

	// Record block number ranges for all chains
	for chainID, ranges := range executingRanges {
		m.m.RecordExecutingBlockRange(
			chainID.String(),
			ranges.min,
			ranges.max,
		)
	}
	for chainID, ranges := range initiatingRanges {
		m.m.RecordInitiatingBlockRange(
			chainID.String(),
			ranges.min,
			ranges.max,
		)
	}

	// Optional read-only cross-check against the interop-filter.
	if m.filterObserver != nil {
		m.filterObserver.Observe(m.ctx, jobMap)
		m.filterObserver.PollFailsafe(m.ctx)
	}

	// Optional read-only observation of each op-supernode.
	for _, obs := range m.supernodeObservers {
		obs.Observe(m.ctx, jobMap)
	}
}
