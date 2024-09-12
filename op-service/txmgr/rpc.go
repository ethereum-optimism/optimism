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

func (a *SimpleTxmgrAPI) GetMinBaseFee(_ context.Context) *big.Int {
	return a.mgr.GetMinBaseFee()
}

func (a *SimpleTxmgrAPI) SetMinBaseFee(_ context.Context, val *big.Int) {
	a.mgr.SetMinBaseFee(val)
}

func (a *SimpleTxmgrAPI) GetMinPriorityFee(_ context.Context) *big.Int {
	return a.mgr.GetMinPriorityFee()
}

func (a *SimpleTxmgrAPI) SetMinPriorityFee(_ context.Context, val *big.Int) {
	a.mgr.SetMinPriorityFee(val)
}

func (a *SimpleTxmgrAPI) GetMinBlobFee(_ context.Context) *big.Int {
	return a.mgr.GetMinBlobFee()
}

func (a *SimpleTxmgrAPI) SetMinBlobFee(_ context.Context, val *big.Int) {
	a.mgr.SetMinBlobFee(val)
}

func (a *SimpleTxmgrAPI) GetFeeThreshold(_ context.Context) *big.Int {
	return a.mgr.GetFeeThreshold()
}

func (a *SimpleTxmgrAPI) SetFeeThreshold(_ context.Context, val *big.Int) {
	a.mgr.SetFeeThreshold(val)
}

func (a *SimpleTxmgrAPI) GetBumpFeeRetryTime(_ context.Context) time.Duration {
	return a.mgr.GetBumpFeeRetryTime()
}

func (a *SimpleTxmgrAPI) SetBumpFeeRetryTime(_ context.Context, val time.Duration) {
	a.mgr.SetBumpFeeRetryTime(val)
}
