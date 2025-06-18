package monitor

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/processors"
	supervisortypes "github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

var ErrNotExecutingMessage = errors.New("not an executing message")

type JobID string

type jobStatus int

const (
	jobStatusUnknown jobStatus = iota
	jobStatusFuture
	jobStatusValid
	jobStatusInvalid
	jobStatusMissing
)

func (j jobStatus) isTerminal() bool {
	switch j {
	case jobStatusValid:
		return true
	case jobStatusInvalid:
		return true
	default:
		return false
	}
}

func (s jobStatus) String() string {
	switch s {
	case jobStatusUnknown:
		return "unknown"
	case jobStatusFuture:
		return "future"
	case jobStatusValid:
		return "valid"
	case jobStatusInvalid:
		return "invalid"
	case jobStatusMissing:
		return "missing"
	default:
		return fmt.Sprintf("unknown status: %d", s)
	}
}

// Job is a job that is being tracked by the monitor
// it represents an executing message and initiating message pair
// it is used to track the status of the job over time
// its getters and setters are thread safe
type Job struct {
	id     JobID
	rwLock sync.RWMutex

	firstSeen     time.Time
	lastEvaluated time.Time
	terminalAt    time.Time
	didMetrics    atomic.Bool

	executingAddress common.Address
	executingChain   eth.ChainID
	executingBlock   eth.BlockID
	executingPayload common.Hash

	initiating     *supervisortypes.Identifier
	initiatingHash []common.Hash

	// track each status seen over time
	status []jobStatus
}

// String returns a string representation of the job
func (j *Job) String() string {
	j.rwLock.RLock()
	defer j.rwLock.RUnlock()
	return fmt.Sprintf("Job{executing: %s@%d:%s, payload: %s, initiating: %s@%d:%d, status: %v}",
		j.executingChain,
		j.executingBlock.Number,
		j.executingBlock.Hash.String()[:10],
		j.executingPayload.String()[:10],
		j.initiating.ChainID,
		j.initiating.BlockNumber,
		j.initiating.LogIndex,
		j.LatestStatus().String())
}

func JobId(
	executingBlockNumber uint64,
	executingLogIndex uint,
	executingPayload common.Hash,
	executingChain eth.ChainID,
	intitiatingBlockNumber uint64,
	logIndex uint32,
	initiatingChain eth.ChainID,
) JobID {
	return JobID(
		fmt.Sprintf(
			"block-%d.%d.%s@chain-%s:block-%d.log-%d@chain-%s",
			executingBlockNumber,
			executingLogIndex,
			executingPayload.String(),
			executingChain.String(),
			intitiatingBlockNumber,
			logIndex,
			initiatingChain.String(),
		))
}

// JobFromExecutingMessageLog converts a log to a job
func JobFromExecutingMessageLog(log *types.Log, executingChain eth.ChainID) (Job, error) {
	msg, err := processors.MessageFromLog(log)
	if err != nil {
		return Job{}, err
	}
	if msg == nil {
		return Job{}, ErrNotExecutingMessage
	}
	return Job{
		id: JobId(
			log.BlockNumber,
			log.Index,
			msg.PayloadHash,
			executingChain,
			msg.Identifier.BlockNumber,
			msg.Identifier.LogIndex,
			msg.Identifier.ChainID,
		),
		executingAddress: log.Address,
		executingChain:   executingChain,
		executingBlock:   eth.BlockID{Hash: log.BlockHash, Number: log.BlockNumber},
		executingPayload: msg.PayloadHash,

		initiating: &msg.Identifier,
	}, nil
}

// BlockReceiptsToJobs converts a slice of receipts to a slice of jobs
func BlockReceiptsToJobs(receipts []*types.Receipt, executingChain eth.ChainID) []*Job {
	jobs := make([]*Job, 0, len(receipts))
	for _, receipt := range receipts {
		for _, log := range receipt.Logs {
			job, err := JobFromExecutingMessageLog(log, executingChain)
			if err != nil {
				continue
			}
			jobs = append(jobs, &job)
		}
	}
	return jobs
}

// ID returns the ID of the job
func (j *Job) ID() JobID {
	j.rwLock.RLock()
	defer j.rwLock.RUnlock()
	return j.id
}

// Statuses returns the states of the job
func (j *Job) Statuses() []jobStatus {
	j.rwLock.RLock()
	defer j.rwLock.RUnlock()

	// Return a copy to prevent external modification
	statuses := make([]jobStatus, len(j.status))
	copy(statuses, j.status)
	return statuses
}

// LatestStatus returns the latest status of the job
func (j *Job) LatestStatus() jobStatus {
	j.rwLock.RLock()
	defer j.rwLock.RUnlock()
	if len(j.status) == 0 {
		return jobStatusUnknown
	}
	return j.status[len(j.status)-1]
}

// TerminalAt returns the time the job last transitioned to a terminal state
func (j *Job) TerminalAt() time.Time {
	j.rwLock.RLock()
	defer j.rwLock.RUnlock()
	return j.terminalAt
}

// UpdateStatus updates the status of the job
func (j *Job) UpdateStatus(status jobStatus) {
	j.rwLock.Lock()
	defer j.rwLock.Unlock()
	// if the job has no status, add the status
	if len(j.status) == 0 {
		j.status = append(j.status, status)
		if status.isTerminal() {
			j.terminalAt = time.Now()
		}
		return
	}
	// if the job status has changed, add the new status
	if j.status[len(j.status)-1] != status {
		j.status = append(j.status, status)
		if status.isTerminal() {
			j.terminalAt = time.Now()
		}
		return
	}
}

// UpdateLastEvaluated updates the last evaluated time of the job
func (j *Job) UpdateLastEvaluated(t time.Time) {
	j.rwLock.Lock()
	defer j.rwLock.Unlock()
	j.lastEvaluated = t
}

// LastEvaluated returns the last evaluated time of the job
func (j *Job) LastEvaluated() time.Time {
	j.rwLock.RLock()
	defer j.rwLock.RUnlock()
	return j.lastEvaluated
}

// DidMetrics returns true if the job has been used to update the metrics at least once
func (j *Job) DidMetrics() bool {
	return j.didMetrics.Load()
}

// SetDidMetrics sets the did metrics flag of the job
func (j *Job) SetDidMetrics() {
	j.didMetrics.Store(true)
}

// AddInitiatingHash adds a hash to the initiatingHash slice if it hasn't been seen before
func (j *Job) AddInitiatingHash(hash common.Hash) {
	j.rwLock.Lock()
	defer j.rwLock.Unlock()

	// Check if latest initiating hash is the same as the hash to be added
	if len(j.initiatingHash) > 0 && j.initiatingHash[len(j.initiatingHash)-1] == hash {
		return
	}

	j.initiatingHash = append(j.initiatingHash, hash)
}

// InitiatingHashes returns a copy of the initiating hashes
func (j *Job) InitiatingHashes() []common.Hash {
	j.rwLock.RLock()
	defer j.rwLock.RUnlock()

	// Return a copy to prevent external modification
	hashes := make([]common.Hash, len(j.initiatingHash))
	copy(hashes, j.initiatingHash)
	return hashes
}
