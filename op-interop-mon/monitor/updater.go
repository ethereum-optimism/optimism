package monitor

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/locks"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	supervisortypes "github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
)

var inboxDepth = 100_000

var ErrLogNotFound = errors.New("log not found")

// TODO: make this configurable
var updateInterval = 1 * time.Second

type UpdaterClient interface {
	FetchReceiptsByNumber(ctx context.Context, number uint64) (eth.BlockInfo, types.Receipts, error)
}

var _ UpdaterClient = &sources.EthClient{}

// Updaters are responsible for updating jobs from a chain for the metric collector to track
type Updater interface {
	Start(ctx context.Context) error
	Enqueue(job *Job)
	Stop() error
	CollectForMetrics(jobs map[JobID]*Job) map[JobID]*Job
}

// RPCFinder connects to an Ethereum chain and extracts receipts in order to create jobs
type RPCUpdater struct {
	client  UpdaterClient
	chainID eth.ChainID

	// the duration after the terminal state is set that the job is considered expired
	expireTime time.Duration

	inbox  chan *Job
	closed chan struct{}

	jobs      sync.Map
	finalized *locks.RWMap[eth.ChainID, eth.NumberAndHash]

	log log.Logger
}

func NewUpdater(
	chainID eth.ChainID,
	client UpdaterClient,
	finalized *locks.RWMap[eth.ChainID, eth.NumberAndHash],
	log log.Logger) *RPCUpdater {
	return &RPCUpdater{
		chainID: chainID,
		client:  client,
		log:     log.New("component", "rpc_updater", "chain_id", chainID),
		// inbox depth is set very deep to allow spikes in job creation plus generous buffer
		inbox:      make(chan *Job, inboxDepth),
		closed:     make(chan struct{}),
		expireTime: 2 * time.Minute,
		finalized:  finalized,
	}
}

func (t *RPCUpdater) Start(ctx context.Context) error {
	go t.Run(ctx)
	return nil
}

func (t *RPCUpdater) Run(ctx context.Context) {
	processTicker := time.NewTicker(updateInterval)
	defer processTicker.Stop()
	defer t.log.Info("updater closed")

	for {
		select {
		case <-t.closed:
			close(t.inbox)
			return
		case job := <-t.inbox:
			t.log.Trace("received job", "job", job.String())
			t.jobs.Store(job.ID(), job)
		case <-processTicker.C:
			t.log.Trace("processing jobs")
			t.processJobs()
			t.log.Trace("processed jobs done")
		}
	}
}

// processJobs handles updating all jobs in the map
func (t *RPCUpdater) processJobs() {
	var toUpdate []*Job
	var toExpire []JobID

	t.jobs.Range(func(key, value any) bool {
		id := key.(JobID)
		job := value.(*Job)
		if t.ShouldExpire(job) {
			t.log.Trace("job should expire", "job", job.String())
			toExpire = append(toExpire, id)
		} else if time.Since(job.LastEvaluated()) >= updateInterval {
			t.log.Trace("job should update", "job", job.String())
			toUpdate = append(toUpdate, job)
		} else {
			t.log.Trace("nothing to do with job", "job", job.String())
		}
		return true
	})

	// Update jobs that need updating
	for _, job := range toUpdate {
		err := t.UpdateJob(job)
		if err != nil {
			t.log.Error("error updating job", "error", err, "job", job.String())
		}
	}

	// Expire jobs that need expiring
	if len(toExpire) > 0 {
		t.expireJobs(toExpire)
	}
}

// expireJobs removes expired jobs from the map
func (t *RPCUpdater) expireJobs(ids []JobID) {
	t.log.Debug("expiring jobs", "ids", ids)

	for _, id := range ids {
		t.jobs.Delete(id)
	}
}

// ShouldExpire returns true if the job should be expired
// jobs should only be expired when *both components* exist in finalized blocks. That is:
// - the initiating block is finalized
// - the executing block is finalized
// Before this point, the job status could change if a reorg affects either the initiating or executing block.
// This also checks that the job has been evaluated at least once, and counted for metrics at least once.
func (t *RPCUpdater) ShouldExpire(job *Job) bool {
	// every job should run at least once, so we can't expire it
	if job.LastEvaluated() == (time.Time{}) {
		t.log.Trace("job has not been evaluated", "job", job.String())
		return false
	}
	// every job should be counted for metrics at least once
	if !job.DidMetrics() {
		t.log.Trace("job has not been counted for metrics", "job", job.String())
		return false
	}
	initExpiryBlock, ok := t.finalized.Get(job.initiating.ChainID)
	if !ok {
		t.log.Warn("initiating chain has no final block", "job", job.String())
		return false
	}
	execExpiryBlock, ok := t.finalized.Get(job.executingChain)
	if !ok {
		t.log.Warn("executing chain has no final block", "job", job.String())
		return false
	}
	if job.initiating.BlockNumber <= initExpiryBlock.NumberU64() &&
		job.executingBlock.Number <= execExpiryBlock.NumberU64() {
		t.log.Debug("job should expire", "job", job.String())
		return true
	} else {
		t.log.Trace("job should not expire", "job", job.String(), "initExpiryBlock", initExpiryBlock.NumberU64(), "execExpiryBlock", execExpiryBlock.NumberU64())
	}
	return false
}

func (t *RPCUpdater) UpdateJob(job *Job) error {
	t.UpdateJobStatus(job)
	job.UpdateLastEvaluated(time.Now())
	t.log.Debug("updated job", "job", job.String())
	return nil
}

func (t *RPCUpdater) UpdateJobStatus(job *Job) {
	blockInfo, receipts, err := t.client.FetchReceiptsByNumber(context.Background(), job.initiating.BlockNumber)
	if err != nil {
		t.log.Error("error getting block receipts", "error", err)
		job.UpdateStatus(jobStatusUnknown)
		return
	}

	// Add the block hash to the job's initiating hashes
	job.AddInitiatingHash(blockInfo.Hash())

	log, err := t.findLogEvent(receipts, job)
	if err == ErrLogNotFound {
		t.log.Error("log not found", "error", err)
		job.UpdateStatus(jobStatusInvalid)
		return
	} else if err != nil {
		t.log.Error("error finding log event", "error", err)
		job.UpdateStatus(jobStatusUnknown)
		return
	}
	// now to confirm the log event matches
	actualHash := crypto.Keccak256Hash(supervisortypes.LogToMessagePayload(log))
	if actualHash != job.executingPayload {
		t.log.Error("log hash mismatch", "expected", job.executingPayload, "got", actualHash)
		job.UpdateStatus(jobStatusInvalid)
		return
	}
	job.UpdateStatus(jobStatusValid)
}

func (t *RPCUpdater) findLogEvent(receipts []*types.Receipt, job *Job) (*types.Log, error) {
	for _, receipt := range receipts {
		for _, log := range receipt.Logs {
			if log.Index == uint(job.initiating.LogIndex) {
				return log, nil
			}
		}
	}
	return nil, ErrLogNotFound
}

// todo: make this a priority queue
func (t *RPCUpdater) Enqueue(job *Job) {
	if t.Stopped() {
		return
	}
	t.inbox <- job
}

// TODO: add wait group to make Stop return sync
func (t *RPCUpdater) Stop() error {
	close(t.closed)
	return nil
}

func (t *RPCUpdater) Stopped() bool {
	select {
	case <-t.closed:
		return true
	default:
		return false
	}
}

// GetJobs adds all jobs to the provided map and returns it
func (t *RPCUpdater) CollectForMetrics(jobs map[JobID]*Job) map[JobID]*Job {
	t.jobs.Range(func(key, value any) bool {
		id := key.(JobID)
		job := value.(*Job)
		job.SetDidMetrics()
		jobs[id] = job
		return true
	})
	return jobs
}
