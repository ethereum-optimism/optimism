package metrics

import (
	ftypes "github.com/HashKeyChain/verse/op-faucet/faucet/backend/types"
	"github.com/HashKeyChain/verse/op-service/eth"
	"github.com/HashKeyChain/verse/op-service/txmgr/metrics"
)

type Metricer interface {
	RecordInfo(version string)
	RecordUp()

	RecordFundAction(faucet ftypes.FaucetID, chainID eth.ChainID, amount eth.ETH) (onDone func(err error))

	metrics.TxMetricer
}
