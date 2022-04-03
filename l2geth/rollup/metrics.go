package rollup

import "github.com/ethereum-optimism/optimism/l2geth/metrics"

var (
	pubTxDropCounter = metrics.NewRegisteredCounter("rollup/pub/txdrops", nil)
)
