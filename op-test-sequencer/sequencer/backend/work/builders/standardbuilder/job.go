package standardbuilder

import (
	"context"
	"sync"

	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/backend/work"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/seqtypes"
)

type Job struct {
	id seqtypes.BuildJobID

	eng apis.BuildAPI

	mu          sync.Mutex
	payloadInfo eth.PayloadInfo
	result      *eth.ExecutionPayloadEnvelope
	unregister  func()
}

func (job *Job) ID() seqtypes.BuildJobID {
	return job.id
}

func (job *Job) Cancel(ctx context.Context) error {
	job.mu.Lock()
	defer job.mu.Unlock()
	err := job.eng.CancelBlock(ctx, job.payloadInfo)
	if err != nil {
		// TODO not-found error is acceptable
		return err
	}
	return nil
}

func (job *Job) Seal(ctx context.Context) (work.Block, error) {
	job.mu.Lock()
	defer job.mu.Unlock()
	envelope, err := job.eng.SealBlock(ctx, job.payloadInfo)
	if err != nil {
		return nil, err
	}
	job.result = envelope
	return envelope, nil
}

func (job *Job) String() string {
	return job.id.String()
}

func (job *Job) Close() {
	job.unregister()
}

var _ work.BuildJob = (*Job)(nil)
