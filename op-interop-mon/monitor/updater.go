package monitor

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	supervisortypes "github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
)

var ErrLogNotFound = errors.New("log not found")

// TODO: make this configurable
var updateInterval = 1 * time.Second

type UpdaterClient interface {
	FetchReceiptsByNumber(ctx context.Context, number uint64) (eth.BlockInfo, types.Receipts, error)
}

var _ UpdaterClient = &sources.EthClient{}

// Updaters are responsible for updating jobs from a chain for the Maintainer to track
type Updater interface {
	Start(ctx context.Context) error
	Enqueue(job *Job)
	Stop() error
	GetJobs(jobs map[JobID]*Job) map[JobID]*Job
}

// RPCFinder connects to an Ethereum chain and extracts receipts in order to create jobs
type RPCUpdater struct {
	client  UpdaterClient
	chainID eth.ChainID

	// the duration after the terminal state is set that the job is considered expired
	expireTime time.Duration

	inbox  chan *Job
	closed chan struct{}

	// Map to track jobs being processed by this updater
	jobMap   map[JobID]*Job
	jobMapMu sync.RWMutex

	log log.Logger
}

func NewUpdater(chainID eth.ChainID, client UpdaterClient, log log.Logger) *RPCUpdater {
	return &RPCUpdater{
		chainID:    chainID,
		client:     client,
		log:        log.New("component", "rpc_updater", "chain_id", chainID),
		inbox:      make(chan *Job, 10_000),
		closed:     make(chan struct{}),
		expireTime: 2 * time.Minute,
		jobMap:     make(map[JobID]*Job),
	}
}

func (t *RPCUpdater) Start(ctx context.Context) error {
	go t.Run(ctx)
	return nil
}

func (t *RPCUpdater) Run(ctx context.Context) {
	// Set up ticker for regular job processing
	processTicker := time.NewTicker(updateInterval)
	defer processTicker.Stop()

	for {
		select {
		case <-t.closed:
			t.log.Info("updater closed")
			close(t.inbox)
			return
		case job := <-t.inbox:
			t.jobMapMu.Lock()
			t.jobMap[job.ID()] = job
			t.jobMapMu.Unlock()
		case <-processTicker.C:
			t.processJobs()
		}
	}
}

// processJobs handles updating all jobs in the map
func (t *RPCUpdater) processJobs() {
	var toUpdate []*Job
	var toExpire []JobID

	t.jobMapMu.RLock()
	for id, job := range t.jobMap {
		if t.ShouldExpire(job) {
			toExpire = append(toExpire, id)
		} else if time.Since(job.lastEvaluated) >= updateInterval {
			toUpdate = append(toUpdate, job)
		}
	}
	t.jobMapMu.RUnlock()

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
	t.jobMapMu.Lock()
	defer t.jobMapMu.Unlock()

	for _, id := range ids {
		if job, ok := t.jobMap[id]; ok {
			t.log.Info("job expired", "job", job.String())
			delete(t.jobMap, id)
		}
	}
}

func (t *RPCUpdater) ShouldExpire(job *Job) bool {
	terminal := job.TerminalAt()
	if terminal == (time.Time{}) {
		return false
	}
	return time.Since(terminal) > t.expireTime
}

func (t *RPCUpdater) UpdateJob(job *Job) error {
	t.UpdateJobStatus(job)
	job.UpdateLastEvaluated(time.Now())
	t.log.Debug("updated job", "job", job.String())
	return nil
}

func (t *RPCUpdater) UpdateJobStatus(job *Job) {
	_, receipts, err := t.client.FetchReceiptsByNumber(context.Background(), job.initiating.BlockNumber)
	if err != nil {
		t.log.Error("error getting block receipts", "error", err)
		job.UpdateStatus(jobStatusUnknown)
		return
	}
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
func (t *RPCUpdater) GetJobs(jobs map[JobID]*Job) map[JobID]*Job {
	t.jobMapMu.RLock()
	defer t.jobMapMu.RUnlock()
	for k, v := range t.jobMap {
		jobs[k] = v
	}
	return jobs
}
