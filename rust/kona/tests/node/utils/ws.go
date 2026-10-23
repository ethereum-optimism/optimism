package node_utils

import (
	"strings"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-service/dial"
	"github.com/ethereum-optimism/optimism/op-service/sources"
)

// RollupWS exercises ordinary rollup RPC methods over the WebSocket transport.
func RollupWS(t devtest.T, node *dsl.L2CLNode) *sources.RollupClient {
	t.Helper()
	endpoint := strings.Replace(node.Escape().UserRPC(), "http", "ws", 1)
	rpc, err := dial.DialRollupClientWithTimeout(t.Ctx(), t.Logger(), endpoint)
	t.Require().NoError(err)
	t.Cleanup(rpc.Close)
	return rpc
}
