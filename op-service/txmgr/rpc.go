package txmgr

import (
	"context"
	"math/big"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
)

func NewTxmgrApi(txmgr TxManager, m metrics.RPCMetricer, l log.Logger) rpc.API {
	return rpc.API{
		Namespace: "txmgr",
		Service: &TxmgrApi{
			mgr: txmgr,
			m:   m,
			l:   l,
		},
	}
}

type TxmgrApi struct {
	mgr TxManager
	m   metrics.RPCMetricer
	l   log.Logger
}

func (t *TxmgrApi) GetMinBaseFee(ctx context.Context) *big.Int {
	recordDur := t.m.RecordRPCServerRequest("txmgr_getMinBaseFee")
	defer recordDur()
	return t.mgr.GetMinBaseFee()
}

func (t *TxmgrApi) SetMinBaseFee(ctx context.Context, val *big.Int) {
	recordDur := t.m.RecordRPCServerRequest("txmgr_setMinBaseFee")
	defer recordDur()
	t.mgr.SetMinBaseFee(val)
}

func (t *TxmgrApi) GetPriorityFee(ctx context.Context) *big.Int {
	recordDur := t.m.RecordRPCServerRequest("txmgr_getPriorityFee")
	defer recordDur()
	return t.mgr.GetPriorityFee()
}

func (t *TxmgrApi) SetPriorityFee(ctx context.Context, val *big.Int) {
	recordDur := t.m.RecordRPCServerRequest("txmgr_setPriorityFee")
	defer recordDur()
	t.mgr.SetPriorityFee(val)
}

func (t *TxmgrApi) GetMinBlobFee(ctx context.Context) *big.Int {
	recordDur := t.m.RecordRPCServerRequest("txmgr_getMinBlobFee")
	defer recordDur()
	return t.mgr.GetMinBlobFee()
}

func (t *TxmgrApi) SetMinBlobFee(ctx context.Context, val *big.Int) {
	recordDur := t.m.RecordRPCServerRequest("txmgr_setMinBlobFee")
	defer recordDur()
	t.mgr.SetMinBlobFee(val)
}

func (t *TxmgrApi) GetFeeThreshold(ctx context.Context) *big.Int {
	recordDur := t.m.RecordRPCServerRequest("txmgr_getFeeThreshold")
	defer recordDur()
	return t.mgr.GetFeeThreshold()
}

func (t *TxmgrApi) SetFeeThreshold(ctx context.Context, val *big.Int) {
	recordDur := t.m.RecordRPCServerRequest("txmgr_setFeeThreshold")
	defer recordDur()
	t.mgr.SetFeeThreshold(val)
}

func (t *TxmgrApi) GetBumpFeeRetryTime(ctx context.Context) time.Duration {
	recordDur := t.m.RecordRPCServerRequest("txmgr_getBumpFeeRetryTime")
	defer recordDur()
	return t.mgr.GetBumpFeeRetryTime()
}

func (t *TxmgrApi) SetBumpFeeRetryTime(ctx context.Context, val time.Duration) {
	recordDur := t.m.RecordRPCServerRequest("txmgr_setBumpFeeRetryTime")
	defer recordDur()
	t.mgr.SetBumpFeeRetryTime(val)
}

func (t *TxmgrApi) GetPendingTxs(ctx context.Context, includeData, includeEncodedBytes bool) ([]PendingTxRPC, error) {
	recordDur := t.m.RecordRPCServerRequest("txmgr_getPendingTxs")
	defer recordDur()
	return t.mgr.GetPendingTxs(includeData, includeEncodedBytes)
}

func (t *TxmgrApi) CancelPendingTx(ctx context.Context, nonce uint64) error {
	recordDur := t.m.RecordRPCServerRequest("txmgr_cancelPendingTx")
	defer recordDur()
	return t.mgr.CancelPendingTx(nonce)
}
