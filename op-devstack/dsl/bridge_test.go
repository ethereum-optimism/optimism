package dsl

import (
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestWithdrawalProvenDisputeGameIndexReturnsCopy(t *testing.T) {
	dt := devtest.SerialT(t)
	stored := big.NewInt(7)
	withdrawal := &Withdrawal{
		commonImpl:   commonFromT(dt),
		proveParams:  ProvenWithdrawalParameters{DisputeGameIndex: stored},
		proveReceipt: &types.Receipt{},
	}

	actual := withdrawal.ProvenDisputeGameIndex()
	require.Equal(t, int64(7), actual.Int64())
	require.NotSame(t, stored, actual)
	actual.SetInt64(8)
	require.Equal(t, int64(7), withdrawal.ProvenDisputeGameIndex().Int64())
}
