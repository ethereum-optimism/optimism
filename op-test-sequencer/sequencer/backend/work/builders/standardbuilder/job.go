package standardbuilder

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/backend/work"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/seqtypes"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
)

type Job struct {
	id seqtypes.BuildJobID

	eng apis.BuildAPI

	attrs     *eth.PayloadAttributes
	parentRef eth.L2BlockRef

	payloadInfo eth.PayloadInfo
	result      *eth.ExecutionPayloadEnvelope
	unregister  func() // always non-nil

	mu     sync.Mutex
	logger log.Logger
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

func (job *Job) Open(ctx context.Context) error {
	job.mu.Lock()
	defer job.mu.Unlock()

	job.logger.Debug("Opening block", "parent_ref", job.parentRef.ID(), "attrs", job.attrs)
	info, err := job.eng.OpenBlock(ctx, job.parentRef.ID(), job.attrs)
	if err != nil {
		return fmt.Errorf("failed to open block: %w", err)
	}
	job.payloadInfo = info
	return nil
}

func (job *Job) Seal(ctx context.Context) (work.Block, error) {
	job.mu.Lock()
	defer job.mu.Unlock()
	if job.result != nil {
		return job.result, nil
	}

	job.logger.Debug("Sealing block", "payload_info", job.payloadInfo)
	envelope, err := job.eng.SealBlock(ctx, job.payloadInfo)
	if err != nil {
		return nil, err
	}
	job.result = envelope

	job.logger.Debug("Sealed block, got envelope", "block_hash", envelope.ExecutionPayload.BlockHash, "parent_hash", envelope.ExecutionPayload.ParentHash)
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

func (job *Job) IncludeTx(ctx context.Context, tx hexutil.Bytes) error {
	job.mu.Lock()
	defer job.mu.Unlock()

	job.attrs.Transactions = append(job.attrs.Transactions, tx)
	return nil
}

var _ work.BuildJob = (*Job)(nil)
