package derive

import (
	"encoding/binary"
	"fmt"
	"math/big"
	"math/rand"
	"testing"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum/go-ethereum/common/hexutil"

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
func GenerateDeposit(source UserDepositSource, rng *rand.Rand) *types.DepositTx {
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
		SourceHash: source.SourceHash(),
		From:       GenerateAddress(rng),
		To:         to,
		Value:      RandETH(rng, 200),
		Gas:        uint64(rng.Int63n(10 * 1e6)), // 10 M gas max
		Data:       data,
		Mint:       mint,
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
	if deposit.Mint != nil {
		deposit.Mint.FillBytes(data[offset : offset+32])
	}
	offset += 32

	deposit.Value.FillBytes(data[offset : offset+32])
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

	return GenerateLog(MockDepositContractAddr, topics, data)
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
			source := UserDepositSource{
				L1BlockHash: randomHash(rng),
				LogIndex:    uint64(rng.Intn(10000)),
			}
			depInput := GenerateDeposit(source, rng)
			log := GenerateDepositLog(depInput)

			log.TxIndex = uint(rng.Intn(10000))
			log.Index = uint(source.LogIndex)
			log.BlockHash = source.L1BlockHash
			depOutput, err := UnmarshalLogEvent(log)
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
	name string
	// generate len(receipts) receipts
	receipts []receiptData
}

func TestDeriveUserDeposits(t *testing.T) {
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
			var receipts []*types.Receipt
			var expectedDeposits []*types.DepositTx
			logIndex := uint(0)
			blockHash := randomHash(rng)
			for txIndex, rData := range testCase.receipts {
				var logs []*types.Log
				status := types.ReceiptStatusSuccessful
				if !rData.goodReceipt {
					status = types.ReceiptStatusFailed
				}
				for _, isDeposit := range rData.DepositLogs {
					var ev *types.Log
					if isDeposit {
						source := UserDepositSource{L1BlockHash: blockHash, LogIndex: uint64(logIndex)}
						dep := GenerateDeposit(source, rng)
						if status == types.ReceiptStatusSuccessful {
							expectedDeposits = append(expectedDeposits, dep)
						}
						ev = GenerateDepositLog(dep)
					} else {
						ev = GenerateLog(GenerateAddress(rng), nil, nil)
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
			got, errs := UserDeposits(receipts, MockDepositContractAddr)
			assert.Equal(t, len(errs), 0)
			assert.Equal(t, len(got), len(expectedDeposits))
			for d, depTx := range got {
				expected := expectedDeposits[d]
				assert.Equal(t, expected, depTx)
			}
		})
	}
}

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
