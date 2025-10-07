package dsl

import (
	"errors"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/retry"
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

type ForkchoiceUpdateResult struct {
	T       devtest.T
	Refresh func()
	Result  *eth.ForkchoiceUpdatedResult
	Err     error
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

func (r *ForkchoiceUpdateResult) WaitUntilValid(attempts int) *ForkchoiceUpdateResult {
	tryCnt := 0
	err := retry.Do0(r.T.Ctx(), attempts, &retry.FixedStrategy{Dur: 1 * time.Second},
		func() error {
			r.Refresh()
			tryCnt += 1
			if r.Result.PayloadStatus.Status != eth.ExecutionValid {
				r.T.Log("Wait for FCU to return valid", "status", r.Result.PayloadStatus.Status, "try_count", tryCnt)
				return errors.New("still syncing")
			}
			return nil
		})
	r.T.Require().NoError(err)
	return r
}
