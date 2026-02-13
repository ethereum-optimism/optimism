package txinclude_test

import (
	"context"
	"errors"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/txinclude"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
)

// mockRPCClient implements txinclude.RPCClient for testing
type mockRPCClient struct {
	Results  [6]hexutil.Bytes
	RPCError error
	Err      error // Non-rpc error. e.g., from the transport layer.
}

func (m *mockRPCClient) BatchCallContext(ctx context.Context, batch []rpc.BatchElem) error {
	if m.Err != nil {
		return m.Err
	}
	if m.RPCError != nil {
		if len(batch) == 0 {
			panic("empty batch")
		}
		batch[0].Error = m.RPCError
	}
	for i, result := range m.Results {
		batch[i].Result = &result
	}
	return nil
}

func TestIsthmusCostOracleSetParams(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := &mockRPCClient{}
		oracle := txinclude.NewIsthmusCostOracle(mock, time.Millisecond)
		require.NoError(t, oracle.SetParams(context.Background()))
	})

	t.Run("RPC error in batch response", func(t *testing.T) {
		mock := &mockRPCClient{
			RPCError: errors.New("the sky is falling"),
		}
		oracle := txinclude.NewIsthmusCostOracle(mock, time.Millisecond)
		require.ErrorIs(t, oracle.SetParams(context.Background()), mock.RPCError)
	})

	t.Run("misc error", func(t *testing.T) {
		mock := &mockRPCClient{
			Err: errors.New("the sky is falling"),
		}
		oracle := txinclude.NewIsthmusCostOracle(mock, time.Millisecond)
		require.ErrorIs(t, oracle.SetParams(context.Background()), mock.Err)
	})
}

func TestIsthmusCostOracleOPCost(t *testing.T) {
	t.Run("account for operator cost", func(t *testing.T) {
		mock := &mockRPCClient{
			Results: [6]hexutil.Bytes{
				// L1 costs are zero.
				hexutil.Bytes{},
				hexutil.Bytes{},
				hexutil.Bytes{},
				hexutil.Bytes{},
				// Operator cost is non-zero.
				big.NewInt(3).Bytes(),
				big.NewInt(4).Bytes(),
			},
		}
		oracle := txinclude.NewIsthmusCostOracle(mock, time.Millisecond)
		require.NoError(t, oracle.SetParams(context.Background()))
		got := oracle.OPCost(types.NewTx(&types.DynamicFeeTx{
			Gas: 2_000_000,
		}))
		require.Equal(t, big.NewInt(10), got, "3 * 2_000_00 / 1_000_000 + 4 = 10")
	})

	t.Run("account for l1 cost", func(t *testing.T) {
		mock := &mockRPCClient{
			Results: [6]hexutil.Bytes{
				// L1 costs are non-zero.
				big.NewInt(102).Bytes(),
				big.NewInt(103).Bytes(),
				big.NewInt(104).Bytes(),
				big.NewInt(105).Bytes(),
				// Operator cost is zero.
				hexutil.Bytes{},
				hexutil.Bytes{},
			},
		}
		oracle := txinclude.NewIsthmusCostOracle(mock, time.Millisecond)
		require.NoError(t, oracle.SetParams(context.Background()))
		tx := types.NewTx(&types.DynamicFeeTx{})
		got := oracle.OPCost(tx)
		want, _ := types.NewL1CostFuncFjord(big.NewInt(102), big.NewInt(104), big.NewInt(103), big.NewInt(105))(tx.RollupCostData())
		require.Equal(t, want, got)
	})
}
