package engine

import (
	"testing"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"

	optypes "github.com/ethereum-optimism/optimism/op-core/types"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

func TestSanityCheckPayload(t *testing.T) {
	deposit := eth.Data{optypes.DepositTxType, 0xc0}
	userTx := eth.Data{types.DynamicFeeTxType, 0xc0}

	tests := []struct {
		name    string
		txs     []eth.Data
		wantErr string
	}{
		{name: "deposits first", txs: []eth.Data{deposit, deposit, userTx}},
		{name: "no transactions", wantErr: "no transactions"},
		{name: "empty first transaction", txs: []eth.Data{{}, userTx}, wantErr: "invalid first transaction"},
		{name: "user transaction first", txs: []eth.Data{userTx, deposit}, wantErr: "first transaction was not deposit tx"},
		{name: "empty later transaction", txs: []eth.Data{deposit, userTx, {}}, wantErr: "failed to decode transaction idx 2"},
		{name: "deposit after user transaction", txs: []eth.Data{deposit, userTx, deposit}, wantErr: "after other tx"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := sanityCheckPayload(&eth.ExecutionPayload{Transactions: tc.txs})
			if tc.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tc.wantErr)
			}
		})
	}
}
