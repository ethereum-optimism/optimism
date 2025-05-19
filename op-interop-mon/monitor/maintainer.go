package monitor

import (
	"context"
	"sync"
	"time"

	"github.com/ethereum-optimism/optimism/op-interop-mon/metrics"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/locks"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
)

// JobUpdater can take cases and update them
type JobUpdater interface {
	UpdateJob(c Job)
}

type receiptClient interface {
	BlockReceipts(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) ([]*types.Receipt, error)
}

type Maintainer struct {
	clients     locks.RWMap[eth.ChainID, receiptClient]
	finders     locks.RWMap[eth.ChainID, Finder]
	updaters    locks.RWMap[eth.ChainID, Updater]
	newInbox    chan *Job
	updateInbox chan *Job
	closed      chan struct{}

	jobMap   map[JobID]*Job
	jobMapMu sync.RWMutex

	log log.Logger
	m   metrics.Metricer
}

func NewMaintainer(log log.Logger, m metrics.Metricer) *Maintainer {
	return &Maintainer{
		newInbox:    make(chan *Job, 10_000),
		updateInbox: make(chan *Job, 10_000),
		jobMap:      make(map[JobID]*Job),
		jobMapMu:    sync.RWMutex{},
		log:         log,
		m:           m,
	}
}

func (m *Maintainer) AddClient(chainID eth.ChainID, client receiptClient) {
	m.clients.Set(chainID, client)
}

func (m *Maintainer) AddFinder(chainID eth.ChainID, finder Finder) {
	m.finders.Set(chainID, finder)
}

func (m *Maintainer) AddUpdater(chainID eth.ChainID, updater Updater) {
	m.updaters.Set(chainID, updater)
}

func (m *Maintainer) Start() error {
	go m.Run()
	return nil
}

// EnqueueNew enqueues a new job
// It will not enqueue a job if it already exists by ID in the jobMap
// and will lock the jobMap while checking and adding the job
func (m *Maintainer) EnqueueNew(c *Job) {
	if m.Stopped() {
		return
	}
	m.jobMapMu.Lock()
	if old, ok := m.jobMap[c.ID()]; ok {
		m.jobMapMu.Unlock()
		m.log.Warn("job with id already exists", "jobID", c.ID(), "job", c, "existing", old.String())
		return
	}
	m.jobMap[c.ID()] = c
	m.jobMapMu.Unlock()
	m.newInbox <- c
}

// EnqueueUpdate enqueues an update job
// the job is expected to already exist in the jobMap
func (m *Maintainer) EnqueueUpdate(c *Job) {
	if m.Stopped() {
		return
	}
	m.updateInbox <- c
}

func (m *Maintainer) Stopped() bool {
	select {
	case <-m.closed:
		return true
	default:
		return false
	}
}

// Run is the main loop for the maintainer
func (m *Maintainer) Run() {
	// set up a ticker to run every 1s
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-m.closed:
			return

		case c := <-m.newInbox:
			m.log.Trace("received new job", "job", c)
			m.ProcessJob(c)
		case c := <-m.updateInbox:
			m.log.Trace("received update job", "job", c)
			m.ProcessJob(c)
		case <-ticker.C:
			m.ConsolidateMetrics()
		}
	}
}

// ProcessJob processes a case
// It mill check if the case is valid, invalid, or missing
// It mill then update the case status and send it back into the inbox
func (m *Maintainer) ProcessJob(c *Job) {
	// the referenced Chain ID is the one mho can update the job
	refChainID := c.initiating.ChainID
	updater, ok := m.updaters.Get(refChainID)
	if !ok {
		m.log.Error("updater not found", "chainID", refChainID)
		return
	}
	updater.Enqueue(c)
}

// ConsolidateMetrics scans the jobMap and updates the metrics
func (m *Maintainer) ConsolidateMetrics() {
	m.jobMapMu.RLock()
	defer m.jobMapMu.RUnlock()
	// message metrics are dimensioned by:
	// - initiating chain id
	// - block number
	// - block hash (only for executing messages)
	// - status
	executingMessages := map[eth.ChainID]map[uint64]map[common.Hash]map[string]int{}
	initiatingMessages := map[eth.ChainID]map[uint64]map[string]int{}
	for _, job := range m.jobMap {
		status := job.LatestStatus().String()
		// Lazy increment the executing message metrics
		if executingMessages[job.executingChain] == nil {
			executingMessages[job.executingChain] = make(map[uint64]map[common.Hash]map[string]int)
		}
		if executingMessages[job.executingChain][job.executingBlock.Number] == nil {
			executingMessages[job.executingChain][job.executingBlock.Number] = make(map[common.Hash]map[string]int)
		}
		if executingMessages[job.executingChain][job.executingBlock.Number][job.executingBlock.Hash] == nil {
			executingMessages[job.executingChain][job.executingBlock.Number][job.executingBlock.Hash] = make(map[string]int)
		}
		if _, ok := executingMessages[job.executingChain][job.executingBlock.Number][job.executingBlock.Hash][status]; !ok {
			executingMessages[job.executingChain][job.executingBlock.Number][job.executingBlock.Hash][status] = 0
		}
		executingMessages[job.executingChain][job.executingBlock.Number][job.executingBlock.Hash][status]++

		// Lazy increment the initiating message metrics
		if initiatingMessages[job.initiating.ChainID] == nil {
			initiatingMessages[job.initiating.ChainID] = make(map[uint64]map[string]int)
		}
		if initiatingMessages[job.initiating.ChainID][job.initiating.BlockNumber] == nil {
			initiatingMessages[job.initiating.ChainID][job.initiating.BlockNumber] = make(map[string]int)
		}
		if _, ok := initiatingMessages[job.initiating.ChainID][job.initiating.BlockNumber][status]; !ok {
			initiatingMessages[job.initiating.ChainID][job.initiating.BlockNumber][status] = 0
		}
		initiatingMessages[job.initiating.ChainID][job.initiating.BlockNumber][status]++
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
}

// TODO: add mait group to make Stop return sync
func (m *Maintainer) Stop() error {
	close(m.closed)
	return nil
}
