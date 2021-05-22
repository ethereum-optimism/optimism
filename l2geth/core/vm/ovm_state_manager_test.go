package vm

import (
	"crypto/rand"
	"math/big"
	"os"
	"sort"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/diffdb"
	"github.com/ethereum/go-ethereum/params"
)

type TestData map[*big.Int]BlockData

// per-block test data are an address + a bunch of k/v pairs
type BlockData map[common.Address][]ContractData

// keys and values are bytes32 in solidity
type ContractData struct {
	key     [32]uint8
	value   [32]uint8
	mutated bool
}

// Test contract addrs
var (
	contract1 = common.HexToAddress("0x000000000000000000000000000000000001")
	contract2 = common.HexToAddress("0x000000000000000000000000000000000002")
)

func makeEnv(dbname string) (*diffdb.DiffDb, *EVM, TestData, *Contract) {
	db, _ := diffdb.NewDiffDb(dbname, 1)
	mock := &mockDb{db: *db}
	env := NewEVM(Context{}, mock, params.TestChainConfig, Config{})
	// re-use `dummyContractRef` from `logger_test.go`
	contract := NewContract(&dummyContractRef{}, &dummyContractRef{}, new(big.Int), 0)
	testData := make(TestData)
	return db, env, testData, contract
}

func TestEthCallNoop(t *testing.T) {
	db, env, _, contract := makeEnv("test1")
	defer os.Remove("test1")
	env.Context.EthCallSender = &common.Address{0}
	env.Context.BlockNumber = big.NewInt(1)
	args := map[string]interface{}{
		"_contract": contract1,
		"_key":      [32]uint8{1},
		"_value":    [32]uint8{2},
	}
	putContractStorage(env, contract, args)
	diff, err := db.GetDiff(env.Context.BlockNumber)
	if err != nil {
		t.Fatal("Db call error", err)
	}
	if len(diff) > 0 {
		t.Fatalf("map must be empty since it was an eth call")
	}
}

func TestSetDiffs(t *testing.T) {
	db, env, testData, contract := makeEnv("test2")
	defer os.Remove("test2")
	// not an eth-call
	env.Context.EthCallSender = nil
	// in block 1 both contracts get touched
	blockNumber := big.NewInt(5)
	testData.addRandomData(blockNumber, contract1, 5)
	testData.addRandomData(blockNumber, contract2, 10)

	// in another block, only 1 contract gets touched
	blockNumber2 := big.NewInt(6)
	testData.addRandomData(blockNumber2, contract2, 10)

	// insert the data in the diffdb via `putContractStorage` calls
	putTestData(t, env, contract, blockNumber, testData)

	// diffs match
	diff, _ := db.GetDiff(blockNumber)
	expected := getExpected(testData[blockNumber])
	if !DiffsEqual(diff, expected) {
		t.Fatalf("Diff did not match.")
	}

	// empty diff for the next block
	diff2, err := db.GetDiff(blockNumber2)
	if err != nil {
		t.Fatal("Db call error", err)
	}
	if len(diff2) != 0 {
		t.Fatalf("Diff2 should be empty since data about the next block is not added yet")
	}

	// insert the data and get the diff again
	putTestData(t, env, contract, blockNumber2, testData)

	expected2 := getExpected(testData[blockNumber2])
	diff2, err = db.GetDiff(blockNumber2)
	if err != nil {
		t.Fatal("Db call error", err)
	}
	if !DiffsEqual(diff2, expected2) {
		t.Fatalf("Diff did not match.")
	}
}

/// Sorted equality between 2 diffs
func DiffsEqual(d1 diffdb.Diff, d2 diffdb.Diff) bool {
	for k, v := range d1 {
		sort.SliceStable(v, func(i, j int) bool {
			return v[i].Key.Big().Cmp(v[j].Key.Big()) < 0
		})

		sort.SliceStable(d2[k], func(i, j int) bool {
			return d2[k][i].Key.Big().Cmp(d2[k][j].Key.Big()) < 0
		})

		exp := d2[k]
		for i, v2 := range v {
			if exp[i] != v2 {
				return false
			}
		}
	}

	return true
}

// inserts a bunch of data for the provided `blockNumber` for all contracts touched in that block
func putTestData(t *testing.T, env *EVM, contract *Contract, blockNumber *big.Int, testData TestData) {
	blockData := testData[blockNumber]
	env.Context.BlockNumber = blockNumber
	for address, data := range blockData {
		for _, contractData := range data {
			args := map[string]interface{}{
				"_contract": address,
				"_key":      contractData.key,
				"_value":    contractData.value,
			}
			_, err := putContractStorage(env, contract, args)
			if err != nil {
				t.Fatalf("Expected nil error, got %s", err)
			}
		}
	}
}

// creates `num` random k/v entries for `contract`'s address at `blockNumber`
func (data TestData) addRandomData(blockNumber *big.Int, contract common.Address, num int) {
	for i := 0; i < num; i++ {
		val := ContractData{
			key:     randBytes(),
			value:   randBytes(),
			mutated: true,
		}

		// alloc empty blockdata
		if data[blockNumber] == nil {
			data[blockNumber] = make(BlockData)
		}
		data[blockNumber][contract] = append(data[blockNumber][contract], val)
	}
}

// the expected diff for the GetDiff call contains the data's keys only, the values & proofs
// are fetched via GetProof
func getExpected(testData BlockData) diffdb.Diff {
	res := make(diffdb.Diff)
	for address, data := range testData {
		for _, contractData := range data {
			key := diffdb.Key{
				Key:     contractData.key,
				Mutated: contractData.mutated,
			}
			res[address] = append(res[address], key)
		}
	}
	return res
}

// creates a random 32 byte array
func randBytes() [32]uint8 {
	bytes := make([]uint8, 32)
	rand.Read(bytes)
	var res [32]uint8
	copy(res[:], bytes)
	return res
}

// Mock everything else
type mockDb struct {
	db diffdb.DiffDb
}

func (mock *mockDb) SetDiffKey(block *big.Int, address common.Address, key common.Hash, mutated bool) error {
	mock.db.SetDiffKey(block, address, key, mutated)
	return nil
}

func (mock *mockDb) SetDiffAccount(block *big.Int, address common.Address) error {
	// mock.db.SetDiffAccount(block, address)
	return nil
}

func (mock *mockDb) CreateAccount(common.Address)                              {}
func (mock *mockDb) SubBalance(common.Address, *big.Int)                       {}
func (mock *mockDb) AddBalance(common.Address, *big.Int)                       {}
func (mock *mockDb) GetBalance(common.Address) *big.Int                        { return big.NewInt(0) }
func (mock *mockDb) GetNonce(common.Address) uint64                            { return 0 }
func (mock *mockDb) SetNonce(common.Address, uint64)                           {}
func (mock *mockDb) GetCodeHash(common.Address) common.Hash                    { return common.Hash{} }
func (mock *mockDb) GetCode(common.Address) []byte                             { return []byte{} }
func (mock *mockDb) SetCode(common.Address, []byte)                            {}
func (mock *mockDb) GetCodeSize(common.Address) int                            { return 0 }
func (mock *mockDb) AddRefund(uint64)                                          {}
func (mock *mockDb) SubRefund(uint64)                                          {}
func (mock *mockDb) GetRefund() uint64                                         { return 0 }
func (mock *mockDb) GetCommittedState(common.Address, common.Hash) common.Hash { return common.Hash{} }
func (mock *mockDb) GetState(common.Address, common.Hash) common.Hash          { return common.Hash{} }
func (mock *mockDb) SetState(common.Address, common.Hash, common.Hash)         {}
func (mock *mockDb) Suicide(common.Address) bool                               { return true }
func (mock *mockDb) HasSuicided(common.Address) bool                           { return true }
func (mock *mockDb) Exist(common.Address) bool                                 { return true }
func (mock *mockDb) Empty(common.Address) bool                                 { return true }
func (mock *mockDb) RevertToSnapshot(int)                                      {}
func (mock *mockDb) Snapshot() int                                             { return 0 }
func (mock *mockDb) AddLog(*types.Log)                                         {}
func (mock *mockDb) AddPreimage(common.Hash, []byte)                           {}
func (mock *mockDb) ForEachStorage(common.Address, func(common.Hash, common.Hash) bool) error {
	return nil
}
