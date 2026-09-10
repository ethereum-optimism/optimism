package atomic

import (
	"context"
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/client"
)

func TestCallTracerLogOrdering(t *testing.T) {
	log := func(value byte, position uint) callLog {
		return callLog{Data: []byte{value}, Position: hexutil.Uint(position)}
	}
	frame := callFrame{
		Logs: []callLog{log(1, 0), log(4, 1), log(5, 2)},
		Calls: []callFrame{
			{Logs: []callLog{log(2, 0), log(3, 1)}, Calls: []callFrame{{Error: "execution reverted", Logs: []callLog{log(99, 0)}}}},
			{Error: "execution reverted", Calls: []callFrame{{Logs: []callLog{log(99, 0)}}}},
		},
	}
	execution, err := executionFromFrame(frame)
	require.NoError(t, err)
	var actual []byte
	for _, log := range execution.Logs {
		actual = append(actual, log.Data...)
	}
	require.Equal(t, []byte{1, 2, 3, 4, 5}, actual)
	frame.Error = "execution reverted"
	execution, err = executionFromFrame(frame)
	require.NoError(t, err)
	require.True(t, execution.Reverted)
	require.Empty(t, execution.Logs)
}

type feeTraceRPC struct {
	client.RPC
	args map[string]any
}

func (r *feeTraceRPC) CallContext(_ context.Context, result any, method string, args ...any) error {
	switch method {
	case "eth_getBlockByHash":
		*result.(**types.Header) = &types.Header{BaseFee: big.NewInt(7)}
	case "debug_traceCall":
		r.args = args[0].(map[string]any)
		*result.(*callFrame) = callFrame{}
	default:
		return fmt.Errorf("unexpected RPC %s", method)
	}
	return nil
}

func TestRPCTransactionFees(t *testing.T) {
	for _, explicit := range []bool{false, true} {
		name := "raw default"
		if explicit {
			name = "pinned fees"
		}
		t.Run(name, func(t *testing.T) {
			rpc := &feeTraceRPC{}
			e := &RPCExecutor{RPC: rpc}
			tx := Transaction{Gas: 100_000}
			if explicit {
				tx.GasFeeCap, tx.GasTipCap = big.NewInt(100), big.NewInt(17)
			}
			_, err := e.Replay(t.Context(), tx)
			require.NoError(t, err)
			if explicit {
				require.NotContains(t, rpc.args, "gasPrice")
				require.Equal(t, tx.GasFeeCap, (*big.Int)(rpc.args["maxFeePerGas"].(*hexutil.Big)))
				require.Equal(t, tx.GasTipCap, (*big.Int)(rpc.args["maxPriorityFeePerGas"].(*hexutil.Big)))
			} else {
				require.Equal(t, big.NewInt(8), (*big.Int)(rpc.args["gasPrice"].(*hexutil.Big)))
				require.NotContains(t, rpc.args, "maxFeePerGas")
				require.NotContains(t, rpc.args, "maxPriorityFeePerGas")
			}
		})
	}
}
