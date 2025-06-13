package monitor

import (
	"time"

	"github.com/ethereum-optimism/optimism/op-interop-mon/metrics"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/locks"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum/go-ethereum/log"
)

type Maintainer struct {
	clients     locks.RWMap[eth.ChainID, *sources.EthClient]
	finders     locks.RWMap[eth.ChainID, Finder]
	updaters    locks.RWMap[eth.ChainID, Updater]
	newInbox    chan *Job
	updateInbox chan *Job
	closed      chan struct{}

	log log.Logger
	m   metrics.Metricer
}

func NewMaintainer(log log.Logger, m metrics.Metricer) *Maintainer {
	return &Maintainer{
		// For ample buffer, we estimate 100k new jobs per second
		// 10k maximum Executing Messages per Block
		// 1 Block per second
		// 10 Chains
		newInbox: make(chan *Job, 100_000),
		// The update inbox has a lower limit so that updaters experience backpressure
		updateInbox: make(chan *Job, 10_000),
		log:         log,
		m:           m,
	}
}

func (m *Maintainer) AddClient(chainID eth.ChainID, client *sources.EthClient) {
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
func (m *Maintainer) EnqueueNew(c *Job) {
	if m.Stopped() {
		return
	}
	m.newInbox <- c
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
	// TODO: these channel waits can cause a deadlock if the updateInbox is full and we
	// are adding new jobs to the updaters. Need a way to offload excess jobs to a buffer,
	// BUT it's not clear how much work this service is likely to be doing in reality.
	updater.Enqueue(c)
}

// TODO: add mait group to make Stop return sync
func (m *Maintainer) Stop() error {
	close(m.closed)
	return nil
}
