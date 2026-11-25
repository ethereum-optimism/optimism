package derive

import (
	"encoding/binary"
	"fmt"
	"math/big"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	optypes "github.com/ethereum-optimism/optimism/op-core/types"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
)

func TestUnmarshalLogEvent(t *testing.T) {
	// t.Skip("not working because deposit_log_create not working properly")
	for i := int64(0); i < 100; i++ {
		t.Run(fmt.Sprintf("random_deposit_%d", i), func(t *testing.T) {
			rng := rand.New(rand.NewSource(1234 + i))
			source := UserDepositSource{
				L1BlockHash: testutils.RandomHash(rng),
				LogIndex:    uint64(rng.Intn(10000)),
			}
			depInput := testutils.GenerateDeposit(source.SourceHash(), rng)
			log, err := MarshalDepositLogEvent(MockDepositContractAddr, depInput)
			if err != nil {
				t.Fatal(err)
			}

			log.TxIndex = uint(rng.Intn(10000))
			log.Index = uint(source.LogIndex)
			log.BlockHash = source.L1BlockHash
			depOutput, err := UnmarshalDepositLogEvent(log)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, depInput, depOutput)
		})
	}
}

// DeriveL1InfoDeposit is tested in reading_test.go, combined with the inverse ParseL1InfoDepositTxData

// receiptData defines what a test receipt looks like
type receiptData struct {
	// false = failed tx
	goodReceipt bool
	// false = not a deposit log
	DepositLogs []bool
}

func makeReceipts(rng *rand.Rand, blockHash common.Hash, depositContractAddr common.Address, testReceipts []receiptData) ([]*types.Receipt, []*optypes.DepositTx, error) {
	logIndex := uint(0)
	receipts := []*types.Receipt{}
	expectedDeposits := []*optypes.DepositTx{}
	for txIndex, rData := range testReceipts {
		var logs []*types.Log
		status := types.ReceiptStatusSuccessful
		if !rData.goodReceipt {
			status = types.ReceiptStatusFailed
		}
		for _, isDeposit := range rData.DepositLogs {
			var ev *types.Log
			var err error
			if isDeposit {
				source := UserDepositSource{L1BlockHash: blockHash, LogIndex: uint64(logIndex)}
				dep := testutils.GenerateDeposit(source.SourceHash(), rng)
				if status == types.ReceiptStatusSuccessful {
					expectedDeposits = append(expectedDeposits, dep)
				}
				ev, err = MarshalDepositLogEvent(depositContractAddr, dep)
				if err != nil {
					return []*types.Receipt{}, []*optypes.DepositTx{}, err
				}
			} else {
				ev = testutils.GenerateLog(testutils.RandomAddress(rng), nil, nil)
			}
			ev.TxIndex = uint(txIndex)
			ev.Index = logIndex
			ev.BlockHash = blockHash
			logs = append(logs, ev)
			logIndex++
		}

		receipts = append(receipts, &types.Receipt{
			Type:             types.DynamicFeeTxType,
			Status:           status,
			Logs:             logs,
			BlockHash:        blockHash,
			TransactionIndex: uint(txIndex),
		})
	}
	return receipts, expectedDeposits, nil
}

type DeriveUserDepositsTestCase struct {
	name string
	// generate len(receipts) receipts
	receipts []receiptData
}

func TestDeriveUserDeposits(t *testing.T) {
	// t.Skip("not working because deposit_log_create not working properly")
	testCases := []DeriveUserDepositsTestCase{
		{"no deposits", []receiptData{}},
		{"other log", []receiptData{{true, []bool{false}}}},
		{"success deposit", []receiptData{{true, []bool{true}}}},
		{"failed deposit", []receiptData{{false, []bool{true}}}},
		{"mixed deposits", []receiptData{{true, []bool{true}}, {false, []bool{true}}}},
		{"success multiple logs", []receiptData{{true, []bool{true, true}}}},
		{"failed multiple logs", []receiptData{{false, []bool{true, true}}}},
		{"not all deposit logs", []receiptData{{true, []bool{true, false, true}}}},
		{"random", []receiptData{{true, []bool{false, false, true}}, {false, []bool{}}, {true, []bool{true}}}},
	}
	for i, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			rng := rand.New(rand.NewSource(1234 + int64(i)))
			blockHash := testutils.RandomHash(rng)
			receipts, expectedDeposits, err := makeReceipts(rng, blockHash, MockDepositContractAddr, testCase.receipts)
			if err != nil {
				t.Fatal(err)
			}
			got, err := UserDeposits(receipts, MockDepositContractAddr)
			require.NoError(t, err)
			require.Equal(t, len(got), len(expectedDeposits))
			for d, depTx := range got {
				expected := expectedDeposits[d]
				require.Equal(t, expected, depTx)
			}
		})
	}
}

// TestUnmarshalDepositLogEventMintAbove128Bits pins the deposit derived from a log whose mint
// needs more than 128 bits. The spec types the mint as uint256, and op-node reads the whole word.
// kona asserts the same log and the same encoded deposit in
// `rust/kona/crates/protocol/protocol/src/deposits.rs` (`test_decode_deposit_mint_above_128_bits`),
// so the two clients derive the same deposit from it.
func TestUnmarshalDepositLogEventMintAbove128Bits(t *testing.T) {
	from := common.HexToAddress("0x1111111111111111111111111111111111111111")
	to := common.HexToAddress("0x2222222222222222222222222222222222222222")
	// 2^128 + 10: both halves of the word are non-zero.
	mint := new(big.Int).Add(new(big.Int).Lsh(big.NewInt(1), 128), big.NewInt(10))

	data := make([]byte, 192)
	data[31] = 32  // opaqueData offset
	data[63] = 128 // opaqueData length
	mint.FillBytes(data[64:96])
	data[127] = 100                                 // value
	binary.BigEndian.PutUint64(data[128:136], 1000) // gas limit
	data[136] = 1                                   // isCreation
	log := &types.Log{
		Topics: []common.Hash{
			DepositEventABIHash,
			common.BytesToHash(from.Bytes()),
			common.BytesToHash(to.Bytes()),
			DepositEventVersion0,
		},
		Data: data,
	}

	dep, err := UnmarshalDepositLogEvent(log)
	require.NoError(t, err)
	require.Equal(t, mint, dep.Mint)
	encoded, err := dep.MarshalBinary()
	require.NoError(t, err)
	require.Equal(t,
		"0x7ef886a0ed428e1c45e1d9561b62834e1a2d3015a0caae3bfdc16b4da059ac885b01a1459411111111111111111111111111111111111111118091010000000000000000000000000000000a648203e880b700000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
		hexutil.Encode(encoded),
	)
}
