package metrics

import (
	ftypes "github.com/HashKeyChain/verse/op-faucet/faucet/backend/types"
	"github.com/HashKeyChain/verse/op-service/eth"
	txmetrics "github.com/HashKeyChain/verse/op-service/txmgr/metrics"
)

type NoopMetrics struct {
	*txmetrics.NoopTxMetrics
}

func (n NoopMetrics) RecordInfo(version string) {}

func (n NoopMetrics) RecordUp() {}

func (n NoopMetrics) RecordFundAction(faucet ftypes.FaucetID, chainID eth.ChainID, amount eth.ETH) (onDone func(err error)) {
	return func(err error) {}
}

var _ Metricer = NoopMetrics{}
