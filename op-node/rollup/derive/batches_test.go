package derive

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
)

type ValidBatchTestCase struct {
	Name      string
	Epoch     rollup.Epoch
	MinL2Time uint64
	MaxL2Time uint64
	Batch     BatchData
	Valid     bool
}

func TestValidBatch(t *testing.T) {
	testCases := []ValidBatchTestCase{
		{
			Name:      "valid epoch",
			Epoch:     123,
			MinL2Time: 43,
			MaxL2Time: 52,
			Batch: BatchData{BatchV1: BatchV1{
				Epoch:        123,
				Timestamp:    43,
				Transactions: []hexutil.Bytes{{0x01, 0x13, 0x37}, {0x02, 0x13, 0x37}},
			}},
			Valid: true,
		},
		{
			Name:      "ignored epoch",
			Epoch:     123,
			MinL2Time: 43,
			MaxL2Time: 52,
			Batch: BatchData{BatchV1: BatchV1{
				Epoch:        122,
				Timestamp:    43,
				Transactions: nil,
			}},
			Valid: false,
		},
		{
			Name:      "too old",
			Epoch:     123,
			MinL2Time: 43,
			MaxL2Time: 52,
			Batch: BatchData{BatchV1: BatchV1{
				Epoch:        123,
				Timestamp:    42,
				Transactions: nil,
			}},
			Valid: false,
		},
		{
			Name:      "too new",
			Epoch:     123,
			MinL2Time: 43,
			MaxL2Time: 52,
			Batch: BatchData{BatchV1: BatchV1{
				Epoch:        123,
				Timestamp:    52,
				Transactions: nil,
			}},
			Valid: false,
		},
		{
			Name:      "wrong time alignment",
			Epoch:     123,
			MinL2Time: 43,
			MaxL2Time: 52,
			Batch: BatchData{BatchV1: BatchV1{
				Epoch:        123,
				Timestamp:    46,
				Transactions: nil,
			}},
			Valid: false,
		},
		{
			Name:      "good time alignment",
			Epoch:     123,
			MinL2Time: 43,
			MaxL2Time: 52,
			Batch: BatchData{BatchV1: BatchV1{
				Epoch:        123,
				Timestamp:    51, // 31 + 2*10
				Transactions: nil,
			}},
			Valid: true,
		},
		{
			Name:      "empty tx",
			Epoch:     123,
			MinL2Time: 43,
			MaxL2Time: 52,
			Batch: BatchData{BatchV1: BatchV1{
				Epoch:        123,
				Timestamp:    43,
				Transactions: []hexutil.Bytes{{}},
			}},
			Valid: false,
		},
		{
			Name:      "sneaky deposit",
			Epoch:     123,
			MinL2Time: 43,
			MaxL2Time: 52,
			Batch: BatchData{BatchV1: BatchV1{
				Epoch:        123,
				Timestamp:    43,
				Transactions: []hexutil.Bytes{{0x01}, {types.DepositTxType, 0x13, 0x37}, {0xc0, 0x13, 0x37}},
			}},
			Valid: false,
		},
	}
	conf := rollup.Config{
		Genesis: rollup.Genesis{
			L2Time: 31, // a genesis time that itself does not align to make it more interesting
		},
		BlockTime: 2,
		// other config fields are ignored and can be left empty.
	}
	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			got := ValidBatch(&testCase.Batch, &conf, testCase.Epoch, testCase.MinL2Time, testCase.MaxL2Time)
			if got != testCase.Valid {
				t.Fatalf("case %v was expected to return %v, but got %v", testCase, testCase.Valid, got)
			}
		})
	}
}
