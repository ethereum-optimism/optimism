package metrics

import (
	ftypes "github.com/ethereum-optimism/optimism/op-faucet/faucet/backend/types"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txmgr/metrics"
)

type Metricer interface {
	RecordInfo(version string)
	RecordUp()

	RecordFundAction(faucet ftypes.FaucetID, chainID eth.ChainID, amount eth.ETH) (onDone func(err error))

	metrics.TxMetricer
}
