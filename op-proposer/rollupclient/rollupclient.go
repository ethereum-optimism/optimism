package rollupclient

import (
	"github.com/ethereum-optimism/optimism/op-node/sources"
	"github.com/ethereum/go-ethereum/rpc"
)

// Deprecated: use sources.RollupClient instead
type RollupClient = sources.RollupClient

// Deprecated: use sources.NewRollupClient instead
func NewRollupClient(rpc *rpc.Client) *sources.RollupClient {
	return sources.NewRollupClient(rpc)
}
