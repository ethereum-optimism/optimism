package base

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"
)

func TestEthSimulateV1(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewMinimal(t)
	ctx := t.Ctx()

	type SimulateParams struct {
		ReturnFullTransactions bool
		BlockStateCalls        []any `json:"blockStateCalls"`
	}

	params := SimulateParams{
		ReturnFullTransactions: true,
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
	// Since the EL will just reuse the l1 attributes tx from the previous block
	// and there is no such transaction for the genesis block).
	sys.L1Network.WaitForBlock()

	// Require the RPC call to succeed
	rpcClient := sys.L2EL.Escape().EthClient().RPC()
	var resp []map[string]any
	err := rpcClient.CallContext(
		ctx,
		&resp,
		"eth_simulateV1",
		params,
		"0x1", // Block 1
	)
	require.NoError(t, err)

	// Require exactly one block, matching input
	require.Len(t, resp, 1)
	respBlock := resp[0]

	// Require exactly one transaction, matching input
	require.Len(t, respBlock["transactions"], 1)
	transaction := (respBlock["transactions"].([]any)[0]).(map[string]any)

	// Transaction type should be dynamic fee transaction type, not a deposit transaction.
	require.Equal(t, "0x2", transaction["type"]) // 0x02 is the dynamic fee transaction type

	// Check Blob Gas Used is nonzero
	// This proves out that eth_simulateV1 can be used to estimate the DA size of a transaction
	bgu, err := hexutil.DecodeUint64(respBlock["blobGasUsed"].(string))
	require.NoError(t, err)
	require.NotZero(t, bgu)

	err = rpcClient.CallContext(
		ctx,
		&resp,
		"eth_simulateV1",
		params,
		"0x0", // Genesis block
	)
	t.Log("resp", resp)
	require.Error(t, err, "eth_simulateV1 cannot be used on the genesis block")
}
