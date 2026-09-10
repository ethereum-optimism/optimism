package atomic

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

func TestEVMTransactionFees(t *testing.T) {
	for _, tc := range []struct {
		name      string
		cap, tip  *big.Int
		price     int64
		wantError bool
	}{
		{name: "raw default", price: 8},
		{name: "explicit priority", cap: big.NewInt(100), tip: big.NewInt(17), price: 24},
		{name: "fee cap binds", cap: big.NewInt(12), tip: big.NewInt(9), price: 12},
		{name: "missing tip", cap: big.NewInt(100), wantError: true},
		{name: "below base fee", cap: big.NewInt(6), tip: big.NewInt(1), wantError: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			st, err := state.New(types.EmptyRootHash, state.NewDatabaseForTesting())
			require.NoError(t, err)
			from, target := common.HexToAddress("0x1000"), common.HexToAddress("0x2000")
			st.SetBalance(from, uint256.NewInt(1e18), tracing.BalanceChangeUnspecified)
			// Return GASPRICE, exposing differences between replay and submitted fees.
			st.SetCode(target, common.FromHex("0x3a60005260206000f3"), tracing.CodeChangeUnspecified)
			e := &EVMExecutor{State: st, Config: params.AllDevChainProtocolChanges, Block: vm.BlockContext{CanTransfer: core.CanTransfer, Transfer: core.Transfer, GetHash: func(uint64) common.Hash { return common.Hash{} }, BlockNumber: big.NewInt(20), Time: 1000, GasLimit: 30_000_000, BaseFee: big.NewInt(7), BlobBaseFee: big.NewInt(1), Difficulty: new(big.Int), Random: &common.Hash{}}}
			result, err := e.Replay(t.Context(), Transaction{From: from, To: target, Gas: 100_000, GasFeeCap: tc.cap, GasTipCap: tc.tip})
			if tc.wantError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.False(t, result.Reverted)
			require.Equal(t, big.NewInt(tc.price), new(big.Int).SetBytes(result.Output))
		})
	}
}
