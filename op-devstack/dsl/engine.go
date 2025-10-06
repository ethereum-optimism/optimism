package dsl

import (
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-service/eth"
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
	T      devtest.T
	Result *eth.ForkchoiceUpdatedResult
	Err    error
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
