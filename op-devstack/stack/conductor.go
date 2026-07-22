package stack

import (
	conductorRpc "github.com/ethereum-optimism/optimism/op-conductor/rpc"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type Conductor interface {
	Common
	ChainID() eth.ChainID

	RpcAPI() conductorRpc.API

	// ProxyRPC exposes the conductor's RPC endpoint for raw calls. With RPC
	// proxying enabled, op-conductor forwards execution (eth_*), rollup
	// (optimism_*), and admin (admin_*) requests to its sequencer while it is
	// the leader, so batchers and proposers can follow the active sequencer.
	ProxyRPC() client.RPC

	// ConsensusEndpoint is the Raft consensus address of this conductor, used
	// when (re-)adding it to a cluster.
	ConsensusEndpoint() string
}
