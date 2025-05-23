package monitor

import (
	"context"
	"errors"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	supervisortypes "github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
)

var ErrLogNotFound = errors.New("log not found")

// TODO: make this configurable
var updateInterval = 1 * time.Second

type UpdaterClient interface {
	BlockReceipts(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) ([]*types.Receipt, error)
}

var _ UpdaterClient = &ethclient.Client{}

// Updaters are responsible for updating jobs from a chain for the Maintainer to track
type Updater interface {
	Start(ctx context.Context) error
	Enqueue(job *Job)
	Stop() error
}

// RPCFinder connects to an Ethereum chain and extracts receipts in order to create jobs
type RPCUpdater struct {
	client  UpdaterClient
	chainID eth.ChainID

	// the duration after the terminal state is set that the job is considered expired
	expireTime time.Duration

	inbox           chan *Job
	enqueueCallback func(*Job)
	expireCallback  func(*Job)
	closed          chan struct{}

	log log.Logger
}

func NewUpdater(chainID eth.ChainID, client UpdaterClient, enqueueCallback func(*Job), expireCallback func(*Job), log log.Logger) *RPCUpdater {
	return &RPCUpdater{
		chainID:         chainID,
		client:          client,
		log:             log.New("component", "rpc_updater", "chain_id", chainID),
		inbox:           make(chan *Job, 10_000),
		enqueueCallback: enqueueCallback,
		expireCallback:  expireCallback,
		closed:          make(chan struct{}),
		expireTime:      1 * time.Minute,
	}
}

func (t *RPCUpdater) Start(ctx context.Context) error {
	go t.Run(ctx)
	return nil
}

func (t *RPCUpdater) Run(ctx context.Context) {
	for {
		select {
		// if the finder is closed, close the inbox and outbox and end the loop
		case <-t.closed:
			t.log.Info("updater closed")
			close(t.inbox)
			return
		// if the inbox has a new job, process the job and send the jobs to the outbox
		case job := <-t.inbox:
			err := t.UpdateJob(job)
			if err != nil {
				t.log.Error("error updating job", "error", err)
				continue
			}
			if t.ShouldExpire(job) {
				t.log.Info("job expired", "job", job.String())
				t.expireCallback(job)
				continue
			}
			t.enqueueCallback(job)
		}
	}
}

func (t *RPCUpdater) ShouldExpire(job *Job) bool {
	if terminal := job.TerminalAt(); terminal != (time.Time{}) {
		return time.Since(terminal) > t.expireTime
	}
	return time.Since(job.lastEvaluated) > t.expireTime
}

func (t *RPCUpdater) UpdateJob(job *Job) error {
	if time.Since(job.lastEvaluated) < updateInterval {
		t.log.Trace("skipping job update", "job", job)
		return nil
	}
	t.UpdateJobStatus(job)
	job.UpdateLastEvaluated(time.Now())
	t.log.Debug("updated job", "job", job.String())
	return nil
}

func (t *RPCUpdater) UpdateJobStatus(job *Job) {
	receipts, err := t.client.BlockReceipts(context.Background(),
		rpc.BlockNumberOrHashWithNumber(rpc.BlockNumber(job.initiating.BlockNumber)))
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
