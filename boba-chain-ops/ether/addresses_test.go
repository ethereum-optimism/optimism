package ether

import (
	"errors"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/bobanetwork/v3-anchorage/boba-bindings/bindings"
	"github.com/bobanetwork/v3-anchorage/boba-bindings/predeploys"
	"github.com/bobanetwork/v3-anchorage/boba-chain-ops/node"
	"github.com/bobanetwork/v3-anchorage/boba-chain-ops/node/nodefakes"
	"github.com/holiman/uint256"
	"github.com/ledgerwatch/erigon-lib/common"
	"github.com/ledgerwatch/erigon-lib/common/hexutil"
	"github.com/ledgerwatch/erigon/core/types"
	"github.com/stretchr/testify/require"
)

type TraceTransactionTest struct {
	name             string
	traceTransaction *node.TraceTransaction
	expected         []*common.Address
}

type MappingTest struct {
	name     string
	input    interface{}
	expected interface{}
}

func TestGetAddressesFromTrace(t *testing.T) {
	traceTransactionType1 := generateTraceTranscation(
		common.HexToAddress("0x4200000000000000000000000000000000000001"),
		common.HexToAddress("0x4200000000000000000000000000000000000002"),
		big.NewInt(100),
	)
	traceTransactionType2 := generateTraceTranscation(
		common.HexToAddress("0x4200000000000000000000000000000000000003"),
		common.HexToAddress("0x4200000000000000000000000000000000000004"),
		big.NewInt(10),
	)
	traceTransactionType3 := generateTraceTranscation(
		common.HexToAddress("0x4200000000000000000000000000000000000005"),
		common.HexToAddress("0x4200000000000000000000000000000000000006"),
		big.NewInt(1),
	)
	traceTransactionType4 := generateTraceTranscation(
		common.HexToAddress("0x4200000000000000000000000000000000000007"),
		common.HexToAddress("0x4200000000000000000000000000000000000008"),
		big.NewInt(1),
	)
	traceTransactionType5 := generateTraceTranscation(
		common.HexToAddress("0x4200000000000000000000000000000000000009"),
		common.HexToAddress("0x4200000000000000000000000000000000000010"),
		big.NewInt(1),
	)
	traceTransactionType6 := generateTraceTranscation(
		common.HexToAddress("0x4200000000000000000000000000000000000011"),
		common.HexToAddress("0x4200000000000000000000000000000000000012"),
		big.NewInt(1),
	)
	traceTransactionType7 := generateTraceTranscation(
		common.HexToAddress("0x4200000000000000000000000000000000000011"),
		common.HexToAddress("0x4200000000000000000000000000000000000012"),
		big.NewInt(0),
	)
	traceTransactionType8 := generateTraceTranscation(
		common.HexToAddress("0x4200000000000000000000000000000000000011"),
		common.HexToAddress("0x4200000000000000000000000000000000000012"),
		big.NewInt(-1),
	)
	tests := make([]*TraceTransactionTest, 5)

	// calls: []
	case1 := &TraceTransactionTest{
		name:             "Test 1",
		traceTransaction: traceTransactionType1,
		expected: []*common.Address{
			&traceTransactionType1.From,
			&traceTransactionType1.From,
			&traceTransactionType1.To,
		},
	}
	tests[0] = case1

	// calls: [calls: []
	case2 := &TraceTransactionTest{
		name:             "Test 2",
		traceTransaction: traceTransactionType2,
		expected: []*common.Address{
			&traceTransactionType2.From,
			&traceTransactionType2.From,
			&traceTransactionType2.To,
		},
	}
	case2.traceTransaction.Calls = []*node.TraceTransaction{
		traceTransactionType1,
	}
	case2.expected = append([]*common.Address{
		&traceTransactionType1.From,
		&traceTransactionType1.To,
	}, case2.expected...)
	tests[1] = case2

	// calls: [calls: [calls: []]]
	case3 := &TraceTransactionTest{
		name:             "Test 3",
		traceTransaction: traceTransactionType3,
		expected: []*common.Address{
			&traceTransactionType3.From,
			&traceTransactionType3.From,
			&traceTransactionType3.To,
		},
	}
	case3.traceTransaction.Calls = []*node.TraceTransaction{
		traceTransactionType2,
	}
	case3.expected = append([]*common.Address{
		&traceTransactionType1.From,
		&traceTransactionType1.To,
		&traceTransactionType2.From,
		&traceTransactionType2.To,
	}, case3.expected...)
	tests[2] = case3

	// calls: [calls:[], calls:[]]
	case4 := &TraceTransactionTest{
		name:             "Test 4",
		traceTransaction: traceTransactionType4,
		expected: []*common.Address{
			&traceTransactionType4.From,
			&traceTransactionType4.From,
			&traceTransactionType4.To,
		},
	}
	case4.traceTransaction.Calls = []*node.TraceTransaction{
		traceTransactionType5,
		traceTransactionType6,
	}
	case4.expected = []*common.Address{
		&traceTransactionType4.From,
		&traceTransactionType4.From,
		&traceTransactionType4.To,
		&traceTransactionType5.From,
		&traceTransactionType5.To,
		&traceTransactionType6.From,
		&traceTransactionType6.To,
	}
	tests[3] = case4

	// invalid payload
	case5 := &TraceTransactionTest{
		name:             "Test 5",
		traceTransaction: traceTransactionType7,
		expected:         []*common.Address{},
	}
	case5.traceTransaction.Calls = []*node.TraceTransaction{
		traceTransactionType8,
	}
	case5.expected = []*common.Address{
		&traceTransactionType7.From,
	}
	tests[4] = case5

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			addresses, err := GetAddressesFromTrace(test.traceTransaction, true)
			require.NoError(t, err)
			require.ElementsMatch(t, addresses, test.expected)
		})
	}
}

func TestLoadAddresses(t *testing.T) {
	crawler := &Crawler{
		Client:             nil,
		EndBlock:           100,
		RpcPollingInterval: 1 * time.Second,
		OutputPath:         "invalid.json",
	}
	_, _, err := crawler.LoadAddresses()
	require.ErrorContains(t, err, "no such file or directory")
	crawler.OutputPath = "./testdata/eth-addresses.json"
	blockNumber, addresses, err := crawler.LoadAddresses()
	require.NoError(t, err)
	require.Equal(t, int64(2), blockNumber)
	address1, address2 := common.HexToAddress("0x4200000000000000000000000000000000000000"), common.HexToAddress("0x4200000000000000000000000000000000000001")
	expectedAddresses := []*common.Address{&address1, &address2}
	require.ElementsMatch(t, addresses, expectedAddresses)
}

func TestSaveAddresses(t *testing.T) {
	crawler := &Crawler{
		Client:             nil,
		EndBlock:           100,
		RpcPollingInterval: 1 * time.Second,
		OutputPath:         "./testdata/test-addresses.json",
	}
	address1, address2 := common.Address{1}, common.Address{2}
	addresses := []*common.Address{&address1, &address2}
	err := crawler.SaveAddresses(2, addresses)
	require.NoError(t, err)
	defer os.Remove("./testdata/test-addresses.json")
	blockNumber, inputAddresses, err := crawler.LoadAddresses()
	require.NoError(t, err)
	require.Equal(t, int64(2), blockNumber)
	require.ElementsMatch(t, addresses, inputAddresses)
}

func TestGetTraceTransaction(t *testing.T) {
	fakeRPC := &nodefakes.FakeRPC{}
	crawler := &Crawler{
		Client:             fakeRPC,
		EndBlock:           100,
		RpcPollingInterval: 1 * time.Second,
		OutputPath:         "invalid.json",
	}
	traceTransaction := generateTraceTranscation(
		common.HexToAddress("0x4200000000000000000000000000000000000001"),
		common.HexToAddress("0x4200000000000000000000000000000000000002"),
		big.NewInt(100),
	)
	to := common.HexToAddress("0x00000000000000000000000000000000deadbeef")
	transaction := &types.LegacyTx{
		GasPrice: uint256.NewInt(0),
		CommonTx: types.CommonTx{
			Gas:   50000,
			To:    &to,
			Value: uint256.NewInt(1),
			R:     *uint256.NewInt(1),
			S:     *uint256.NewInt(1),
			V:     *uint256.NewInt(1),
		},
	}
	txHash := transaction.Hash()
	block := node.Block{Number: 1, GasLimit: 1000000, Transactions: []*common.Hash{&txHash}}

	fakeRPC.GetBlockByNumberReturns(&block, nil)
	traceResult, err := crawler.GetTraceTransaction(common.Big1)
	require.NoError(t, err)
	require.Nil(t, traceResult)

	fakeRPC.TraceTransactionReturns(traceTransaction, nil)
	traceResult, err = crawler.GetTraceTransaction(common.Big1)
	require.NoError(t, err)
	require.Equal(t, traceTransaction, traceResult)

	fakeErr := errors.New("error")
	fakeRPC.TraceTransactionReturns(nil, fakeErr)
	_, err = crawler.GetTraceTransaction(common.Big1)
	require.ErrorIs(t, err, fakeErr)
}

func TestMapAddresses(t *testing.T) {
	tests := []MappingTest{
		{
			name: "Test 1",
			input: []*common.Address{
				{1},
				{2},
			},
			expected: map[common.Address]bool{
				{1}: true,
				{2}: true,
			},
		},
		{
			name: "Test 2",
			input: []*common.Address{
				{1},
				{1},
			},
			expected: map[common.Address]bool{
				{1}: true,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input, ok := test.input.([]*common.Address)
			require.Equal(t, true, ok)
			expected, ok := test.expected.(map[common.Address]bool)
			require.Equal(t, true, ok)
			result := MapAddresses(input)
			require.Equal(t, expected, result)
		})
	}
}

func TestAddAddressesToMap(t *testing.T) {
	type inputStruct struct {
		addresses  []*common.Address
		addressMap map[common.Address]bool
	}
	tests := []MappingTest{
		{
			name: "Test 1",
			input: inputStruct{
				addresses: []*common.Address{
					{1},
					{2},
				},
				addressMap: map[common.Address]bool{
					{1}: true,
				},
			},
			expected: map[common.Address]bool{
				{1}: true,
				{2}: true,
			},
		},
		{
			name: "Test 2",
			input: inputStruct{
				addresses: []*common.Address{
					{1},
					{1},
				},
				addressMap: map[common.Address]bool{
					{1}: true,
				},
			},
			expected: map[common.Address]bool{
				{1}: true,
			},
		},
	}
	for _, test := range tests {
		input, ok := test.input.(inputStruct)
		require.Equal(t, true, ok)
		expected, ok := test.expected.(map[common.Address]bool)
		require.Equal(t, true, ok)
		t.Run(test.name, func(t *testing.T) {
			AddAddressesToMap(input.addresses, input.addressMap)
			require.Equal(t, expected, input.addressMap)
		})

	}
}

func TestMapToAddresses(t *testing.T) {
	tests := []MappingTest{
		{
			name: "Test 1",
			input: map[common.Address]bool{
				{1}: true,
				{2}: true,
			},
			expected: []*common.Address{
				{1},
				{2},
			},
		},
	}
	for _, test := range tests {
		input, ok := test.input.(map[common.Address]bool)
		require.Equal(t, true, ok)
		expected, ok := test.expected.([]*common.Address)
		require.Equal(t, true, ok)
		t.Run(test.name, func(t *testing.T) {
			result := MapToAddresses(input)
			require.ElementsMatch(t, expected, result)
		})
	}
}

func TestGetToFromEthMintLogs(t *testing.T) {
	LegacyERC20ETHMetaData := bindings.MetaData{
		ABI: bindings.LegacyERC20ETHABI,
		Bin: bindings.LegacyERC20ETHBin,
	}
	ABI, err := LegacyERC20ETHMetaData.GetAbi()
	require.NoError(t, err)
	fakeRPC := &nodefakes.FakeRPC{}
	crawler := &Crawler{
		Client:             fakeRPC,
		EndBlock:           100,
		RpcPollingInterval: 1 * time.Second,
		OutputPath:         "invalid.json",
	}
	tests := []MappingTest{
		{
			name: "Test 1",
			input: []*types.Log{
				{
					Address: predeploys.LegacyERC20ETHAddr,
					Topics: []common.Hash{
						ABI.Events["Transfer"].ID,
						common.BytesToHash(common.Address{1}.Bytes()),
						common.BytesToHash(common.Address{2}.Bytes()),
					},
				},
			},
			expected: []*common.Address{
				{2},
				{1},
			},
		},
		{
			name: "Test 2",
			input: []*types.Log{
				{
					Address: predeploys.LegacyERC20ETHAddr,
					Topics: []common.Hash{
						ABI.Events["Transfer"].ID,
						{0},
						common.BytesToHash(common.Address{1}.Bytes()),
					},
				},
			},
			expected: []*common.Address{
				{1},
			},
		},
		{
			name: "Test 3",
			input: []*types.Log{
				{
					Address: predeploys.LegacyERC20ETHAddr,
					Topics: []common.Hash{
						ABI.Events["Transfer"].ID,
						common.BytesToHash(common.Address{1}.Bytes()),
						common.BytesToHash(common.Address{2}.Bytes()),
					},
				},
				{
					Address: predeploys.LegacyERC20ETHAddr,
					Topics: []common.Hash{
						ABI.Events["Transfer"].ID,
						common.BytesToHash(common.Address{2}.Bytes()),
						common.BytesToHash(common.Address{1}.Bytes()),
					},
				},
			},
			expected: []*common.Address{
				{2},
				{1},
				{1},
				{2},
			},
		},
		{
			name: "Test 4",
			input: []*types.Log{
				{
					Address: predeploys.LegacyERC20ETHAddr,
					Topics: []common.Hash{
						ABI.Events["Transfer"].ID,
						{1},
					},
				},
			},
			expected: []*common.Address(nil),
		},
		{
			name: "Test 5",
			input: []*types.Log{
				{
					Address: predeploys.LegacyERC20ETHAddr,
					Topics: []common.Hash{
						{1},
						{2},
						{3},
					},
				},
			},
			expected: []*common.Address(nil),
		},
	}
	for i, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input, ok := test.input.([]*types.Log)
			if !ok {
				t.Errorf("Test %d: Invalid input", i)
			}
			expected, ok := test.expected.([]*common.Address)
			require.Equal(t, true, ok)
			fakeRPC.GetLogsReturns(input, nil)
			result, err := crawler.GetToFromEthMintLogs(common.Big0)
			require.NoError(t, err)
			require.Equal(t, expected, result)
		})
	}

}

func TestCheckEthSlots(t *testing.T) {
	alloc := types.GenesisAlloc{
		predeploys.LegacyERC20ETHAddr: types.GenesisAccount{
			Storage: map[common.Hash]common.Hash{},
		},
	}
	crawler := &Crawler{
		Client:             nil,
		EndBlock:           100,
		RpcPollingInterval: 1 * time.Second,
		OutputPath:         "./testdata/eth-addresses.json",
	}
	_, addresses, err := crawler.LoadAddresses()
	require.NoError(t, err)
	for _, addr := range addresses {
		alloc[predeploys.LegacyERC20ETHAddr].Storage[CalcOVMETHStorageKey(*addr)] = common.Hash{1}
	}
	err = CheckEthSlots(alloc, "./testdata/eth-addresses.json")
	require.NoError(t, err)

	commonStorageKey := []common.Hash{
		common.BytesToHash([]byte{2}),
		common.BytesToHash([]byte{3}),
		common.BytesToHash([]byte{4}),
		common.BytesToHash([]byte{5}),
		common.BytesToHash([]byte{6}),
	}
	for _, slot := range commonStorageKey {
		alloc[predeploys.LegacyERC20ETHAddr].Storage[slot] = common.Hash{1}
	}
	err = CheckEthSlots(alloc, "./testdata/eth-addresses.json")
	require.NoError(t, err)

	alloc[predeploys.LegacyERC20ETHAddr].Storage[CalcOVMETHStorageKey(common.Address{1})] = common.Hash{1}
	err = CheckEthSlots(alloc, "./testdata/eth-addresses.json")
	require.ErrorContains(t, err, "not valid")
}

func generateTraceTranscation(from common.Address, to common.Address, value *big.Int) *node.TraceTransaction {
	return &node.TraceTransaction{
		From:  from,
		To:    to,
		Value: hexutil.Big(*value),
	}
}
