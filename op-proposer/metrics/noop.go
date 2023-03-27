package metrics

import (
	"github.com/ethereum-optimism/optimism/op-node/eth"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum/go-ethereum/core/types"
)

type noopMetrics struct{ opmetrics.NoopRefMetrics }

var NoopMetrics Metricer = new(noopMetrics)

func (*noopMetrics) RecordInfo(version string) {}
func (*noopMetrics) RecordUp()                 {}

func (*noopMetrics) RecordL2BlocksProposed(l2ref eth.L2BlockRef) {}
func (*noopMetrics) RecordL1GasFee(receipt *types.Receipt) {
}
