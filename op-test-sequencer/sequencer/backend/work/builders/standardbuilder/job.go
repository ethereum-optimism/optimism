package standardbuilder

import (
	"context"
	"errors"
	"sync"

	"github.com/ethereum/go-ethereum/rpc"

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
	unregister  func() // always non-nil
}

func (job *Job) ID() seqtypes.BuildJobID {
	return job.id
}

func (job *Job) Cancel(ctx context.Context) error {
	job.mu.Lock()
	defer job.mu.Unlock()
	err := job.eng.CancelBlock(ctx, job.payloadInfo)
	if err != nil {
		var rpcErr rpc.Error
		if errors.As(err, &rpcErr) && eth.ErrorCode(rpcErr.ErrorCode()) == eth.UnknownPayload {
			// This error is acceptable, as there is nothing to cancel
			return nil
		}
		return err
	}
	return nil
}

func (job *Job) Seal(ctx context.Context) (work.Block, error) {
	job.mu.Lock()
	defer job.mu.Unlock()
	if job.result != nil {
		return job.result, nil
	}
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
	job.mu.Lock()
	defer job.mu.Unlock()
	job.unregister()
}

var _ work.BuildJob = (*Job)(nil)
