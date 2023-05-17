package metrics

import (
	"math/big"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	txmetrics "github.com/ethereum-optimism/optimism/op-service/txmgr/metrics"
	common "github.com/ethereum/go-ethereum/common"
)

type noopMetrics struct {
	opmetrics.NoopRefMetrics
	txmetrics.NoopTxMetrics
}

var NoopMetrics Metricer = new(noopMetrics)

func (*noopMetrics) RecordInfo(version string) {}
func (*noopMetrics) RecordUp()                 {}

func (*noopMetrics) RecordL2BlocksProposed(l2ref eth.L2BlockRef)                           {}
func (*noopMetrics) RecordValidOutputAlreadyProposed(block *big.Int, output common.Hash)   {}
func (*noopMetrics) RecordInvalidOutputAlreadyProposed(block *big.Int, output common.Hash) {}
