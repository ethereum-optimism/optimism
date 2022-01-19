package l2

import (
	"encoding/binary"
	"fmt"
	"math/big"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func GenerateAddress(rng *rand.Rand) (out common.Address) {
	rng.Read(out[:])
	return
}

func RandETH(rng *rand.Rand, max int64) *big.Int {
	x := big.NewInt(rng.Int63n(max))
	x = new(big.Int).Mul(x, big.NewInt(1e18))
	return x
}

// Returns a DepositEvent customized on the basis of the id parameter.
func GenerateDeposit(blockNum uint64, txIndex uint64, rng *rand.Rand) *types.DepositTx {
	dataLen := rng.Int63n(10_000)
	data := make([]byte, dataLen)
	rng.Read(data)

	var to *common.Address
	if rng.Intn(2) == 0 {
		x := GenerateAddress(rng)
		to = &x
	}
	var mint *big.Int
	if rng.Intn(2) == 0 {
		mint = RandETH(rng, 200)
	}

	dep := &types.DepositTx{
		BlockHeight:      blockNum,
		TransactionIndex: txIndex,
		From:             GenerateAddress(rng),
		To:               to,
		Value:            RandETH(rng, 200),
		Gas:              uint64(rng.Int63n(10 * 1e6)), // 10 M gas max
		Data:             data,
		Mint:             mint,
	}
	return dep
}

// Generates an EVM log entry that encodes a TransactionDeposited event from the deposit contract.
// Calls GenerateDeposit with random number generator to generate the deposit.
func GenerateDepositLog(deposit *types.DepositTx) *types.Log {

	toBytes := common.Hash{}
	if deposit.To != nil {
		toBytes = deposit.To.Hash()
	}
	topics := []common.Hash{
		DepositEventABIHash,
		deposit.From.Hash(),
		toBytes,
	}

	data := make([]byte, 6*32)
	offset := 0
	deposit.Value.FillBytes(data[offset : offset+32])
	offset += 32

	if deposit.Mint != nil {
		deposit.Mint.FillBytes(data[offset : offset+32])
	}
	offset += 32

	binary.BigEndian.PutUint64(data[offset+24:offset+32], deposit.Gas)
	offset += 32
	if deposit.To == nil { // isCreation
		data[offset+31] = 1
	}
	offset += 32
	binary.BigEndian.PutUint64(data[offset+24:offset+32], 5*32)
	offset += 32
	binary.BigEndian.PutUint64(data[offset+24:offset+32], uint64(len(deposit.Data)))
	data = append(data, deposit.Data...)
	if len(data)%32 != 0 { // pad to multiple of 32
		data = append(data, make([]byte, 32-(len(data)%32))...)
	}

	return GenerateLog(DepositContractAddr, topics, data)
}

// Generates an EVM log entry with the given topics and data.
func GenerateLog(addr common.Address, topics []common.Hash, data []byte) *types.Log {
	return &types.Log{
		Address: addr,
		Topics:  topics,
		Data:    data,
		Removed: false,

		// ignored (zeroed):
		BlockNumber: 0,
		TxHash:      common.Hash{},
		TxIndex:     0,
		BlockHash:   common.Hash{},
		Index:       0,
	}
}

func TestUnmarshalLogEvent(t *testing.T) {
	for i := int64(0); i < 100; i++ {
		t.Run(fmt.Sprintf("random_deposit_%d", i), func(t *testing.T) {
			rng := rand.New(rand.NewSource(1234 + i))
			blockNum := rng.Uint64()
			txIndex := uint64(rng.Intn(10000))
			depInput := GenerateDeposit(blockNum, txIndex, rng)
			log := GenerateDepositLog(depInput)
			depOutput, err := UnmarshalLogEvent(blockNum, txIndex, log)
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

type DeriveUserDepositsTestCase struct {
	name   string
	height uint64
	// generate len(receipts) receipts
	receipts []receiptData
}

func TestDeriveUserDeposits(t *testing.T) {
	testCases := []DeriveUserDepositsTestCase{
		{"no deposits", 100, []receiptData{}},
		{"other log", 100, []receiptData{{true, []bool{false}}}},
		{"success deposit", 100, []receiptData{{true, []bool{true}}}},
		{"failed deposit", 100, []receiptData{{false, []bool{true}}}},
		{"mixed deposits", 100, []receiptData{{true, []bool{true}}, {false, []bool{true}}}},
		{"success multiple logs", 100, []receiptData{{true, []bool{true, true}}}},
		{"failed multiple logs", 100, []receiptData{{false, []bool{true, true}}}},
		{"not all deposit logs", 100, []receiptData{{true, []bool{true, false, true}}}},
		{"random", 100, []receiptData{{true, []bool{false, false, true}}, {false, []bool{}}, {true, []bool{true}}}},
	}
	for i, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			rng := rand.New(rand.NewSource(1234 + int64(i)))
			var receipts []*types.Receipt
			var expectedDeposits []*types.DepositTx
			for _, rData := range testCase.receipts {
				var logs []*types.Log
				status := types.ReceiptStatusSuccessful
				if !rData.goodReceipt {
					status = types.ReceiptStatusFailed
				}
				for _, isDeposit := range rData.DepositLogs {
					if isDeposit {
						dep := GenerateDeposit(testCase.height, uint64(1+len(expectedDeposits)), rng)
						if status == types.ReceiptStatusSuccessful {
							expectedDeposits = append(expectedDeposits, dep)
						}
						logs = append(logs, GenerateDepositLog(dep))
					} else {
						logs = append(logs, GenerateLog(GenerateAddress(rng), nil, nil))
					}
				}

				receipts = append(receipts, &types.Receipt{
					Type:   types.DynamicFeeTxType,
					Status: status,
					Logs:   logs,
				})
			}
			got, err := DeriveUserDeposits(testCase.height, receipts)
			assert.NoError(t, err)
			assert.Equal(t, len(got), len(expectedDeposits))
			for d, depTx := range got {
				expected := expectedDeposits[d]
				assert.Equal(t, expected, depTx)
			}
		})
	}
}
