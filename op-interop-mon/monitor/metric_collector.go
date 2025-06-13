package monitor

import (
	"time"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

type InteropMessageMetrics interface {
	RecordExecutingMessageStats(chainID string, blockNumber uint64, blockHash string, status string, value float64)
	RecordInitiatingMessageStats(chainID string, blockNumber uint64, status string, value float64)
	RecordTerminalStatusChange(executingChainID string, initiatingChainID string, value float64)
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

// Run is the main loop for the maintainer
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
	// - initiating chain id
	// - block number
	// - block hash (only for executing messages)
	// - status
	executingMessages := map[eth.ChainID]map[uint64]map[common.Hash]map[string]int{}
	initiatingMessages := map[eth.ChainID]map[uint64]map[string]int{}
	terminalStatusChanges := map[eth.ChainID]map[eth.ChainID]int{}
	for _, job := range jobMap {
		statuses := job.Statuses()
		if len(statuses) == 0 {
			m.log.Warn("Job has no statuses", "job", job)
			continue
		}
		current := statuses[len(statuses)-1].String()
		// Lazy increment the executing message metrics
		if _, ok := executingMessages[job.executingChain]; !ok {
			executingMessages[job.executingChain] = make(map[uint64]map[common.Hash]map[string]int)
		}
		if _, ok := executingMessages[job.executingChain][job.executingBlock.Number]; !ok {
			executingMessages[job.executingChain][job.executingBlock.Number] = make(map[common.Hash]map[string]int)
		}
		if _, ok := executingMessages[job.executingChain][job.executingBlock.Number][job.executingBlock.Hash]; !ok {
			executingMessages[job.executingChain][job.executingBlock.Number][job.executingBlock.Hash] = make(map[string]int)
		}
		if _, ok := executingMessages[job.executingChain][job.executingBlock.Number][job.executingBlock.Hash][current]; !ok {
			executingMessages[job.executingChain][job.executingBlock.Number][job.executingBlock.Hash][current] = 0
		}
		executingMessages[job.executingChain][job.executingBlock.Number][job.executingBlock.Hash][current]++

		// Lazy increment the initiating message metrics
		if _, ok := initiatingMessages[job.initiating.ChainID]; !ok {
			initiatingMessages[job.initiating.ChainID] = make(map[uint64]map[string]int)
		}
		if _, ok := initiatingMessages[job.initiating.ChainID][job.initiating.BlockNumber]; !ok {
			initiatingMessages[job.initiating.ChainID][job.initiating.BlockNumber] = make(map[string]int)
		}
		if _, ok := initiatingMessages[job.initiating.ChainID][job.initiating.BlockNumber][current]; !ok {
			initiatingMessages[job.initiating.ChainID][job.initiating.BlockNumber][current] = 0
		}
		initiatingMessages[job.initiating.ChainID][job.initiating.BlockNumber][current]++

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
			m.log.Warn("Job has been both valid and invalid",
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
	// executing messages
	for chainID, blockNumberMap := range executingMessages {
		for blockNumber, blockHashMap := range blockNumberMap {
			for blockHash, statusMap := range blockHashMap {
				for status, count := range statusMap {
					m.log.Info("updating executing message stats", "chainID", chainID, "blockNumber", blockNumber, "blockHash", blockHash, "status", status, "count", count)
					m.m.RecordExecutingMessageStats(chainID.String(), blockNumber, blockHash.String(), status, float64(count))
				}
			}
		}
	}
	// initiating messages
	for chainID, blockNumberMap := range initiatingMessages {
		for blockNumber, statusMap := range blockNumberMap {
			for status, count := range statusMap {
				m.log.Info("updating initiating message stats", "chainID", chainID, "blockNumber", blockNumber, "status", status, "count", count)
				m.m.RecordInitiatingMessageStats(chainID.String(), blockNumber, status, float64(count))
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
}
