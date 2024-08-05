package txmgr

import (
	"context"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/log"
)

type SimpleTxmgrAPI struct {
	mgr *SimpleTxManager
	l   log.Logger
}

func (a *SimpleTxmgrAPI) GetMinBaseFee(ctx context.Context) *big.Int {
	return a.mgr.GetMinBaseFee()
}

func (a *SimpleTxmgrAPI) SetMinBaseFee(ctx context.Context, val *big.Int) {
	a.mgr.SetMinBaseFee(val)
}

func (a *SimpleTxmgrAPI) GetPriorityFee(ctx context.Context) *big.Int {
	return a.mgr.GetPriorityFee()
}

func (a *SimpleTxmgrAPI) SetPriorityFee(ctx context.Context, val *big.Int) {
	a.mgr.SetPriorityFee(val)
}

func (a *SimpleTxmgrAPI) GetMinBlobFee(ctx context.Context) *big.Int {
	return a.mgr.GetMinBlobFee()
}

func (a *SimpleTxmgrAPI) SetMinBlobFee(ctx context.Context, val *big.Int) {
	a.mgr.SetMinBlobFee(val)
}

func (a *SimpleTxmgrAPI) GetFeeThreshold(ctx context.Context) *big.Int {
	return a.mgr.GetFeeThreshold()
}

func (a *SimpleTxmgrAPI) SetFeeThreshold(ctx context.Context, val *big.Int) {
	a.mgr.SetFeeThreshold(val)
}

func (a *SimpleTxmgrAPI) GetBumpFeeRetryTime(ctx context.Context) time.Duration {
	return a.mgr.GetBumpFeeRetryTime()
}

func (a *SimpleTxmgrAPI) SetBumpFeeRetryTime(ctx context.Context, val time.Duration) {
	a.mgr.SetBumpFeeRetryTime(val)
}
