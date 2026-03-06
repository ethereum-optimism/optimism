package stack

import (
	conductorRpc "github.com/ethereum-optimism/optimism/op-conductor/rpc"
)

type Conductor interface {
	Common
	ID() ComponentID

	RpcAPI() conductorRpc.API
}
