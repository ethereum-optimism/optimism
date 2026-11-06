package dsl

import (
	"errors"
	"fmt"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/retry"
	"github.com/ethereum/go-ethereum/rpc"
)

type NewPayloadResult struct {
	T      devtest.T
	Status *eth.PayloadStatusV1
	Err    error
}

func (r *NewPayloadResult) IsPayloadStatus(status eth.ExecutePayloadStatus) *NewPayloadResult {
	r.T.Require().NotNil(r.Status, "payload status nil")
	r.T.Require().Equal(status, r.Status.Status)
	return r
}

func (r *NewPayloadResult) IsSyncing() *NewPayloadResult {
	r.IsPayloadStatus(eth.ExecutionSyncing)
	r.T.Require().NoError(r.Err)
	return r
}

func (r *NewPayloadResult) IsValid() *NewPayloadResult {
	r.IsPayloadStatus(eth.ExecutionValid)
	r.T.Require().NoError(r.Err)
	return r
}

func (r *NewPayloadResult) IsInvalid() *NewPayloadResult {
	r.IsPayloadStatus(eth.ExecutionInvalid)
	r.T.Require().NoError(r.Err)
	return r
}

type ForkchoiceUpdateResult struct {
	T          devtest.T
	Refresh    func()
	Result     *eth.ForkchoiceUpdatedResult
	ValidCnt   int // count for VALID response
	SyncingCnt int // count for SYNCING response
	InvalidCnt int // count for INVALID response
	RefreshCnt int
	Err        error
}

func (r *ForkchoiceUpdateResult) IsForkchoiceUpdatedStatus(status eth.ExecutePayloadStatus) *ForkchoiceUpdateResult {
	r.T.Require().NotNil(r.Result, "fcu result nil")
	r.T.Require().Equal(status, r.Result.PayloadStatus.Status)
	return r
}

func (r *ForkchoiceUpdateResult) IsSyncing() *ForkchoiceUpdateResult {
	r.IsForkchoiceUpdatedStatus(eth.ExecutionSyncing)
	r.T.Require().NoError(r.Err)
	return r
}

func (r *ForkchoiceUpdateResult) IsValid() *ForkchoiceUpdateResult {
	r.IsForkchoiceUpdatedStatus(eth.ExecutionValid)
	r.T.Require().NoError(r.Err)
	return r
}

// IsInvalidForkchoiceState requires the update to have been refused with the engine API's
// -38002 error and its "Invalid forkchoice state" message, which an execution engine answers
// when a nonzero safe or finalized hash is not on the head's chain. A refused update carries no
// payload status.
func (r *ForkchoiceUpdateResult) IsInvalidForkchoiceState() *ForkchoiceUpdateResult {
	r.T.Require().Nil(r.Result, "a forkchoice update with an invalid state must be refused with an error, not answered with a status")
	var rpcErr rpc.Error
	r.T.Require().ErrorAs(r.Err, &rpcErr, "expected an engine error code, got: %v", r.Err)
	r.T.Require().Equal(eth.InvalidForkchoiceState, eth.ErrorCode(rpcErr.ErrorCode()), "unexpected engine error: %v", r.Err)
	r.T.Require().EqualError(rpcErr, "Invalid forkchoice state")
	return r
}

func (r *ForkchoiceUpdateResult) WaitUntilValid(attempts int) *ForkchoiceUpdateResult {
	tryCnt := 0
	err := retry.Do0(r.T.Ctx(), attempts, &retry.FixedStrategy{Dur: 1 * time.Second},
		func() error {
			r.Refresh()
			tryCnt += 1
			if r.Err != nil {
				return fmt.Errorf("forkchoice returned error: %w", r.Err)
			}
			if r.Result == nil {
				return errors.New("forkchoice has empty result")
			}
			if r.Result.PayloadStatus.Status != eth.ExecutionValid {
				r.T.Logger().Info("Wait for FCU to return valid", "status", r.Result.PayloadStatus.Status, "try_count", tryCnt)
				return errors.New("still syncing")
			}
			return nil
		})
	r.T.Require().NoError(err)
	return r
}

func (r *ForkchoiceUpdateResult) Retry(attempts int) *ForkchoiceUpdateResult {
	tryCnt := 0
	err := retry.Do0(r.T.Ctx(), attempts, &retry.FixedStrategy{Dur: 500 * time.Millisecond},
		func() error {
			r.Refresh()
			tryCnt += 1
			if r.Err != nil {
				return fmt.Errorf("forkchoice returned error: %w", r.Err)
			}
			if r.Result == nil {
				return errors.New("forkchoice has empty result")
			}
			r.T.Logger().Info("Retrying FCU", "status", r.Result.PayloadStatus.Status, "try_count", tryCnt)
			return errors.New("retry")
		})
	r.T.Require().Error(err) // always return error for retrying
	return r
}

func (r *ForkchoiceUpdateResult) ResultAllSyncing() {
	r.T.Require().Equal(r.RefreshCnt, r.SyncingCnt)
}
