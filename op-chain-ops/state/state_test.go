package state_test

import (
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/ethereum-optimism/optimism/op-bindings/solc"
	"github.com/ethereum-optimism/optimism/op-chain-ops/state"
	"github.com/ethereum-optimism/optimism/op-chain-ops/state/testdata"

	"github.com/stretchr/testify/require"
)

var (
	// layout is the storage layout used in tests
	layout solc.StorageLayout
	// testKey is the same test key that geth uses
	testKey, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	// chainID is the chain id used for simulated backends
	chainID = big.NewInt(1337)
)

// Read the test data from disk asap
func init() {
	data, err := os.ReadFile("./testdata/layout.json")
	if err != nil {
		panic("layout.json not found")

	}
	if err := json.Unmarshal(data, &layout); err != nil {
		panic("cannot unmarshal storage layout")
	}
}

func TestSetAndGetStorageSlots(t *testing.T) {
	values := state.StorageValues{}
	values["_uint256"] = new(big.Int).SetUint64(0xafff_ffff_ffff_ffff)
	values["_address"] = common.HexToAddress("0xEA674fdDe714fd979de3EdF0F56AA9716B898ec8")
	values["_bool"] = true
	values["offset0"] = uint8(0xaa)
	values["offset1"] = uint8(0xbb)
	values["offset2"] = uint16(0x0c0c)
	values["offset3"] = uint32(0xf33d35)
	values["offset4"] = uint64(0xd34dd34d00)
	values["offset5"] = new(big.Int).SetUint64(0x43ad0043ad0043ad)
	values["_bytes32"] = common.Hash{0xff}
	values["_string"] = "foobar"

	addresses := make(map[any]any)
	addresses[big.NewInt(1)] = common.Address{19: 0xff}

	values["addresses"] = addresses

	slots, err := state.ComputeStorageSlots(&layout, values)
	require.Nil(t, err)

	backend := backends.NewSimulatedBackend(
		core.GenesisAlloc{
			crypto.PubkeyToAddress(testKey.PublicKey): {Balance: big.NewInt(10000000000000000)},
		},
		15000000,
	)
	opts, err := bind.NewKeyedTransactorWithChainID(testKey, chainID)
	require.Nil(t, err)

	_, _, contract, err := testdata.DeployTestdata(opts, backend)
	require.Nil(t, err)
	backend.Commit()

	// Call each of the methods to make sure that they are set to their 0 values
	testContractStateValuesAreEmpty(t, contract)

	// Send transactions through the set storage API on the contract
	for _, slot := range slots {
		_, err := contract.SetStorage(opts, slot.Key, slot.Value)
		require.Nil(t, err)
	}
	backend.Commit()

	testContractStateValuesAreSet(t, contract, values)

	// Call the get storage API on the contract to double check
	// that the storage slots have been set correctly
	for _, slot := range slots {
		value, err := contract.GetStorage(&bind.CallOpts{}, slot.Key)
		require.Nil(t, err)
		require.Equal(t, value[:], slot.Value.Bytes())
	}
}

// Ensure that all the storage variables are set after setting storage
// through the contract
func testContractStateValuesAreSet(t *testing.T, contract *testdata.Testdata, values state.StorageValues) {
OUTER:
	for key, value := range values {
		var res any
		var err error
		switch key {
		case "_uint256":
			res, err = contract.Uint256(&bind.CallOpts{})
		case "_address":
			res, err = contract.Address(&bind.CallOpts{})
		case "_bool":
			res, err = contract.Bool(&bind.CallOpts{})
		case "offset0":
			res, err = contract.Offset0(&bind.CallOpts{})
		case "offset1":
			res, err = contract.Offset1(&bind.CallOpts{})
		case "offset2":
			res, err = contract.Offset2(&bind.CallOpts{})
		case "offset3":
			res, err = contract.Offset3(&bind.CallOpts{})
		case "offset4":
			res, err = contract.Offset4(&bind.CallOpts{})
		case "offset5":
			res, err = contract.Offset5(&bind.CallOpts{})
		case "_bytes32":
			res, err = contract.Bytes32(&bind.CallOpts{})
			result, ok := res.([32]uint8)
			require.Equal(t, ok, true)
			require.Nil(t, err)
			require.Equal(t, common.BytesToHash(result[:]), value)
			continue OUTER
		case "_string":
			res, err = contract.String(&bind.CallOpts{})
		case "addresses":
			addrs, ok := value.(map[any]any)
			require.Equal(t, ok, true)
			for mapKey, mapVal := range addrs {
				res, err = contract.Addresses(&bind.CallOpts{}, mapKey.(*big.Int))
				require.Nil(t, err)
				require.Equal(t, mapVal, res)
				continue OUTER
			}
		default:
			require.Fail(t, fmt.Sprintf("Unknown variable label: %s", key))
		}
		require.Nil(t, err)
		require.Equal(t, res, value)
	}
}

func testContractStateValuesAreEmpty(t *testing.T, contract *testdata.Testdata) {
	addr, err := contract.Address(&bind.CallOpts{})
	require.Nil(t, err)
	require.Equal(t, addr, common.Address{})

	boolean, err := contract.Bool(&bind.CallOpts{})
	require.Nil(t, err)
	require.Equal(t, boolean, false)

	uint256, err := contract.Uint256(&bind.CallOpts{})
	require.Nil(t, err)
	require.Equal(t, uint256.Uint64(), uint64(0))

	offset0, err := contract.Offset0(&bind.CallOpts{})
	require.Nil(t, err)
	require.Equal(t, offset0, uint8(0))

	offset1, err := contract.Offset1(&bind.CallOpts{})
	require.Nil(t, err)
	require.Equal(t, offset1, uint8(0))

	offset2, err := contract.Offset2(&bind.CallOpts{})
	require.Nil(t, err)
	require.Equal(t, offset2, uint16(0))

	offset3, err := contract.Offset3(&bind.CallOpts{})
	require.Nil(t, err)
	require.Equal(t, offset3, uint32(0))

	offset4, err := contract.Offset4(&bind.CallOpts{})
	require.Nil(t, err)
	require.Equal(t, offset4, uint64(0))

	offset5, err := contract.Offset5(&bind.CallOpts{})
	require.Nil(t, err)
	require.Equal(t, offset5.Uint64(), uint64(0))

	bytes32, err := contract.Bytes32(&bind.CallOpts{})
	require.Nil(t, err)
	require.Equal(t, common.BytesToHash(bytes32[:]), common.Hash{})
}

func TestMergeStorage(t *testing.T) {
	cases := []struct {
		input  []*state.EncodedStorage
		expect []*state.EncodedStorage
	}{
		{
			// One input should be the same result
			input: []*state.EncodedStorage{
				{
					Key:   common.Hash{},
					Value: common.Hash{},
				},
			},
			expect: []*state.EncodedStorage{
				{
					Key:   common.Hash{},
					Value: common.Hash{},
				},
			},
		},
		{
			// Two duplicate inputs should be merged
			input: []*state.EncodedStorage{
				{
					Key:   common.Hash{1},
					Value: common.Hash{},
				},
				{
					Key:   common.Hash{1},
					Value: common.Hash{},
				},
			},
			expect: []*state.EncodedStorage{
				{
					Key:   common.Hash{1},
					Value: common.Hash{},
				},
			},
		},
		{
			// Two different inputs should be the same result
			input: []*state.EncodedStorage{
				{
					Key:   common.Hash{1},
					Value: common.Hash{},
				},
				{
					Key:   common.Hash{2},
					Value: common.Hash{},
				},
			},
			expect: []*state.EncodedStorage{
				{
					Key:   common.Hash{1},
					Value: common.Hash{},
				},
				{
					Key:   common.Hash{2},
					Value: common.Hash{},
				},
			},
		},
		{
			// Two matching keys should be merged bitwise
			input: []*state.EncodedStorage{
				{
					Key:   common.Hash{},
					Value: common.Hash{0x00, 0x01},
				},
				{
					Key:   common.Hash{},
					Value: common.Hash{0x02, 0x00},
				},
			},
			expect: []*state.EncodedStorage{
				{
					Key:   common.Hash{},
					Value: common.Hash{0x02, 0x01},
				},
			},
		},
	}

	for _, test := range cases {
		got := state.MergeStorage(test.input)
		require.Equal(t, test.expect, got)
	}
}

func TestEncodeUintValue(t *testing.T) {
	cases := []struct {
		number any
		offset uint
		expect common.Hash
	}{
		{
			number: 0,
			offset: 0,
			expect: common.Hash{},
		},
		{
			number: big.NewInt(1),
			offset: 0,
			expect: common.Hash{31: 0x01},
		},
		{
			number: uint64(2),
			offset: 0,
			expect: common.Hash{31: 0x02},
		},
		{
			number: uint8(3),
			offset: 0,
			expect: common.Hash{31: 0x03},
		},
		{
			number: uint16(4),
			offset: 0,
			expect: common.Hash{31: 0x04},
		},
		{
			number: uint32(5),
			offset: 0,
			expect: common.Hash{31: 0x05},
		},
		{
			number: int(6),
			offset: 0,
			expect: common.Hash{31: 0x06},
		},
		{
			number: 1,
			offset: 1,
			expect: common.Hash{30: 0x01},
		},
		{
			number: 1,
			offset: 10,
			expect: common.Hash{21: 0x01},
		},
	}

	for _, test := range cases {
		got, err := state.EncodeUintValue(test.number, test.offset)
		require.Nil(t, err)
		require.Equal(t, got, test.expect)
	}
}

func TestEncodeBoolValue(t *testing.T) {
	cases := []struct {
		boolean any
		offset  uint
		expect  common.Hash
	}{
		{
			boolean: true,
			offset:  0,
			expect:  common.Hash{31: 0x01},
		},
		{
			boolean: false,
			offset:  0,
			expect:  common.Hash{},
		},
		{
			boolean: true,
			offset:  1,
			expect:  common.Hash{30: 0x01},
		},
		{
			boolean: false,
			offset:  1,
			expect:  common.Hash{},
		},
		{
			boolean: "true",
			offset:  0,
			expect:  common.Hash{31: 0x01},
		},
		{
			boolean: "false",
			offset:  0,
			expect:  common.Hash{},
		},
	}

	for _, test := range cases {
		got, err := state.EncodeBoolValue(test.boolean, test.offset)
		require.Nil(t, err)
		require.Equal(t, got, test.expect)
	}
}

func TestEncodeAddressValue(t *testing.T) {
	cases := []struct {
		addr   any
		offset uint
		expect common.Hash
	}{
		{
			addr:   common.Address{},
			offset: 0,
			expect: common.Hash{},
		},
		{
			addr:   common.Address{0x01},
			offset: 0,
			expect: common.Hash{12: 0x01},
		},
		{
			addr:   "0x829BD824B016326A401d083B33D092293333A830",
			offset: 0,
			expect: common.HexToHash("0x829BD824B016326A401d083B33D092293333A830"),
		},
		{
			addr:   common.Address{19: 0x01},
			offset: 1,
			expect: common.Hash{30: 0x01},
		},
		{
			addr:   &common.Address{},
			offset: 0,
			expect: common.Hash{},
		},
	}

	for _, test := range cases {
		got, err := state.EncodeAddressValue(test.addr, test.offset)
		require.Nil(t, err)
		require.Equal(t, got, test.expect)
	}
}

func TestEncodeBytes32Value(t *testing.T) {
	cases := []struct {
		bytes32 any
		expect  common.Hash
	}{
		{
			bytes32: common.Hash{0xff},
			expect:  common.Hash{0xff},
		},
		{
			bytes32: "0x11ffffff00000000000000000000000000000000000000000000000000000000",
			expect:  common.HexToHash("0x11ffffff00000000000000000000000000000000000000000000000000000000"),
		},
	}

	for _, test := range cases {
		got, err := state.EncodeBytes32Value(test.bytes32, 0)
		require.Nil(t, err)
		require.Equal(t, got, test.expect)
	}
}

func TestEncodeStringValue(t *testing.T) {
	cases := []struct {
		str    any
		expect common.Hash
	}{
		{
			str:    "foo",
			expect: common.Hash{0x66, 0x6f, 0x6f, 31: 6},
		},
		// Taken from mainnet WETH at 0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2
		{
			str:    "Wrapped Ether",
			expect: common.HexToHash("0x577261707065642045746865720000000000000000000000000000000000001a"),
		},
		{
			str:    "WETH",
			expect: common.HexToHash("0x5745544800000000000000000000000000000000000000000000000000000008"),
		},
	}

	for _, test := range cases {
		got, err := state.EncodeStringValue(test.str, 0)
		require.Nil(t, err)
		require.Equal(t, got, test.expect)
	}
}
