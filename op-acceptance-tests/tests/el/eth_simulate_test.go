package el

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/stretchr/testify/require"
)

func TestEthSimulateV1(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewMinimal(t)
	ctx := t.Ctx()

	type SimulateParams struct {
		BlockStateCalls []any `json:"blockStateCalls"`
	}

	params := SimulateParams{
		BlockStateCalls: []any{
			map[string]any{
				"calls": []any{
					map[string]any{
						"from": "0x0000000000000000000000000000000000000000",
						"to":   "0x0000000000000000000000000000000000000000",
						"data": "0x",
					},
				},
			},
		},
	}

	// wait until the chain mines at least one block
	// (known limitation that we cannot simulate on top of the genesis block,
	// since the EL will just reuse the l1 attributes tx from the previous bock
	// and there is no such transaction for the genesis block).
	//
	sys.L1Network.WaitForBlock()

	rpcClient := sys.L2EL.Escape().EthClient().RPC()
	var resp []map[string]any
	err := rpcClient.CallContext(
		ctx,
		&resp,
		"eth_simulateV1",
		params,
	)
	require.NoError(t, err)
}
