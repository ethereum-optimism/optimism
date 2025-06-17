package monitor

import (
	"time"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/log"
)

type InteropMessageMetrics interface {
	RecordMessageStatus(executingChainID string, initiatingChainID string, status string, count float64)
	RecordTerminalStatusChange(executingChainID string, initiatingChainID string, count float64)
	RecordExecutingBlockRange(chainID string, min uint64, max uint64)
	RecordInitiatingBlockRange(chainID string, min uint64, max uint64)
}

type MetricCollector struct {
	updaters map[eth.ChainID]Updater

	closed chan struct{}
	log    log.Logger
	m      InteropMessageMetrics
}

func NewMetricCollector(log log.Logger, m InteropMessageMetrics, updaters map[eth.ChainID]Updater) *MetricCollector {
	return &MetricCollector{
		log:      log,
		m:        m,
		updaters: updaters,
		closed:   make(chan struct{}),
	}
}

func (m *MetricCollector) Start() error {
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
	return nil
}

// CollectMetrics scans the jobMaps, consolidates them, and updates the metrics
func (m *MetricCollector) CollectMetrics() {
	jobMap := map[JobID]*Job{}
	for _, updater := range m.updaters {
		jobMap = updater.CollectForMetrics(jobMap)
	}
	// message metrics are dimensioned by:
	// - executing chain id
	// - initiating chain id
	// - status
	messageStatus := map[eth.ChainID]map[eth.ChainID]map[string]int{}
	terminalStatusChanges := map[eth.ChainID]map[eth.ChainID]int{}

	// Track block number ranges for each chain
	executingRanges := map[eth.ChainID]struct {
		min, max uint64
	}{}
	initiatingRanges := map[eth.ChainID]struct {
		min, max uint64
	}{}

	for _, job := range jobMap {
		// Initialize executing ranges for this chain if not seen before
		if _, exists := executingRanges[job.executingChain]; !exists {
			executingRanges[job.executingChain] = struct {
				min, max uint64
			}{
				min: job.executingBlock.Number,
				max: job.executingBlock.Number,
			}
		}

		// Initialize initiating ranges for this chain if not seen before
		if _, exists := initiatingRanges[job.initiating.ChainID]; !exists {
			initiatingRanges[job.initiating.ChainID] = struct {
				min, max uint64
			}{
				min: job.initiating.BlockNumber,
				max: job.initiating.BlockNumber,
			}
		}

		// Update executing ranges
		execRange := executingRanges[job.executingChain]
		if job.executingBlock.Number < execRange.min {
			execRange.min = job.executingBlock.Number
		}
		if job.executingBlock.Number > execRange.max {
			execRange.max = job.executingBlock.Number
		}
		executingRanges[job.executingChain] = execRange

		// Update initiating ranges
		initRange := initiatingRanges[job.initiating.ChainID]
		if job.initiating.BlockNumber < initRange.min {
			initRange.min = job.initiating.BlockNumber
		}
		if job.initiating.BlockNumber > initRange.max {
			initRange.max = job.initiating.BlockNumber
		}
		initiatingRanges[job.initiating.ChainID] = initRange

		statuses := job.Statuses()
		if len(statuses) == 0 {
			m.log.Warn("Job has no statuses", "job", job)
			continue
		}
		current := statuses[len(statuses)-1].String()

		// Log invalid statuses
		if current == jobStatusInvalid.String() {
			m.log.Warn("Invalid Executing Message Detected",
				"executing_chain_id", job.executingChain,
				"initiating_chain_id", job.initiating.ChainID,
				"executing_block_height", job.executingBlock.Number,
				"initiating_block_height", job.initiating.BlockNumber,
				"executing_block_hash", job.executingBlock.Hash,
			)
		}

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
		}

		// Lazy increment the message status metrics
		if _, ok := messageStatus[job.executingChain]; !ok {
			messageStatus[job.executingChain] = make(map[eth.ChainID]map[string]int)
		}
		if _, ok := messageStatus[job.executingChain][job.initiating.ChainID]; !ok {
			messageStatus[job.executingChain][job.initiating.ChainID] = make(map[string]int)
		}
		if _, ok := messageStatus[job.executingChain][job.initiating.ChainID][current]; !ok {
			messageStatus[job.executingChain][job.initiating.ChainID][current] = 0
		}
		messageStatus[job.executingChain][job.initiating.ChainID][current]++

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
			if _, ok := terminalStatusChanges[job.executingChain]; !ok {
				terminalStatusChanges[job.executingChain] = make(map[eth.ChainID]int)
			}
			if _, ok := terminalStatusChanges[job.executingChain][job.initiating.ChainID]; !ok {
				terminalStatusChanges[job.executingChain][job.initiating.ChainID] = 0
			}
			terminalStatusChanges[job.executingChain][job.initiating.ChainID]++
		}
	}
	// now we have the metrics consolidated, we can update the metrics
	// message status
	for executingChainID, initiatingChainMap := range messageStatus {
		for initiatingChainID, statusMap := range initiatingChainMap {
			for status, count := range statusMap {
				if status == jobStatusInvalid.String() {
					// invalid messages are logged as warnings
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
	// terminal status changes
	for chainID, initiatingChainIDMap := range terminalStatusChanges {
		for initiatingChainID, count := range initiatingChainIDMap {
			m.m.RecordTerminalStatusChange(
				chainID.String(),
				initiatingChainID.String(),
				float64(count),
			)
		}
	}

	// Record block number ranges
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
}
