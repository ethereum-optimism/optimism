package metrics

import (
	"math/big"

	eth "github.com/ethereum-optimism/optimism/op-node/eth"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	txmetrics "github.com/ethereum-optimism/optimism/op-service/txmgr/metrics"
	common "github.com/ethereum/go-ethereum/common"
	types "github.com/ethereum/go-ethereum/core/types"
)

type noopMetrics struct {
	opmetrics.NoopRefMetrics
	txmetrics.NoopTxMetrics
}

var NoopMetrics Metricer = new(noopMetrics)

func (*noopMetrics) RecordInfo(version string) {}
func (*noopMetrics) RecordUp()                 {}

func (*noopMetrics) RecordValidOutput(l2ref eth.L2BlockRef)                             {}
func (*noopMetrics) RecordInvalidOutput(l2ref eth.L2BlockRef)                           {}
func (*noopMetrics) RecordChallengeSent(l2BlockNumber *big.Int, outputRoot common.Hash) {}
func (*noopMetrics) RecordL1GasFee(receipt *types.Receipt)                              {}
func (*noopMetrics) RecordDisputeGameCreated(l2BlockNumber *big.Int, outputRoot common.Hash, contract common.Address) {
}
