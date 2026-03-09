package stack

import (
	conductorRpc "github.com/ethereum-optimism/optimism/op-conductor/rpc"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type Conductor interface {
	Common
	ChainID() eth.ChainID

	RpcAPI() conductorRpc.API
}
