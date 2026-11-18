package fees

import (
	"encoding/binary"
	"math"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	optypes "github.com/ethereum-optimism/optimism/op-core/types"
)

// The vectors in this file are the values op-geth computes for the same inputs, and op-revm
// computes the same. op-revm has no counterpart of the errors.

var l1CostBedrockVectors = []struct {
	rollupDataGas               uint64
	l1BaseFee, overhead, scalar *big.Int
	want                        string
}{
	{1618, big.NewInt(1000 * 1e6), big.NewInt(50), big.NewInt(7 * 1e6), "11676000000000"},
	{0, big.NewInt(1), big.NewInt(0), big.NewInt(1_000_000), "0"},
	{0, big.NewInt(30e9), big.NewInt(188), big.NewInt(684000), "3857760000000"},
	{530, big.NewInt(2300000), big.NewInt(1000), big.NewInt(684000), "2406996000"},
	{1_000_000, big.NewInt(500e9), big.NewInt(2100), big.NewInt(1_000_000_000), "501050000000000000000"},
}

func TestL1CostBedrockGoldenVectors(t *testing.T) {
	for _, v := range l1CostBedrockVectors {
		got := L1CostBedrock(v.rollupDataGas, v.l1BaseFee, v.overhead, v.scalar)
		require.Equal(t, v.want, got.String(), "rollupDataGas=%d", v.rollupDataGas)
	}
}

var (
	unitL1FeeParams = L1FeeParams{L1BaseFee: big.NewInt(1), L1BlobBaseFee: big.NewInt(1), BaseFeeScalar: big.NewInt(1), BlobFeeScalar: big.NewInt(1)}
	// gweiL1FeeParams has gwei-scale L1 fees and six-digit scalars.
	gweiL1FeeParams = L1FeeParams{L1BaseFee: big.NewInt(7 * 1e9), L1BlobBaseFee: big.NewInt(5 * 1e8), BaseFeeScalar: big.NewInt(684000), BlobFeeScalar: big.NewInt(810949)}
	// maxScalarL1FeeParams has the largest scalars the L1Block contract can hold.
	maxScalarL1FeeParams = L1FeeParams{L1BaseFee: big.NewInt(500e9), L1BlobBaseFee: big.NewInt(100e9), BaseFeeScalar: big.NewInt(math.MaxUint32), BlobFeeScalar: big.NewInt(math.MaxUint32)}
)

var l1CostFjordVectors = []struct {
	fastLzSize uint64
	params     L1FeeParams
	want       string
}{
	{0, L1FeeParams{L1BaseFee: fjordBaseFee, L1BlobBaseFee: fjordBlobBaseFee, BaseFeeScalar: fjordBaseFeeScalar, BlobFeeScalar: fjordBlobBaseFeeScalar}, "3203000"},
	{0, unitL1FeeParams, "0"},
	{0, gweiL1FeeParams, "7701347450000"},
	{0, maxScalarL1FeeParams, "3478923508950000000"},
	{171, L1FeeParams{L1BaseFee: fjordBaseFee, L1BlobBaseFee: fjordBlobBaseFee, BaseFeeScalar: fjordBaseFeeScalar, BlobFeeScalar: fjordBlobBaseFeeScalar}, "3217602"},
	{171, maxScalarL1FeeParams, "3494783921227303050"},
	{500, gweiL1FeeParams, "28931220689957"},
	{12345, gweiL1FeeParams, "792007103150974"},
	{12345, maxScalarL1FeeParams, "357772733706140117550"},
	{130_000, unitL1FeeParams, "1"},
	{130_000, maxScalarL1FeeParams, "3781673849357850088800"},
}

func TestL1CostFjordGoldenVectors(t *testing.T) {
	for _, v := range l1CostFjordVectors {
		got := L1CostFjord(RollupCostData{FastLzSize: v.fastLzSize}, v.params)
		require.Equal(t, v.want, got.String(), "fastLzSize=%d params=%+v", v.fastLzSize, v.params)
	}
}

type txRollupCostDataVector struct {
	name string
	tx   *types.Transaction
	want RollupCostData
}

// txRollupCostDataTestKey signs the signed transactions of this file. Signatures are
// deterministic (RFC 6979), so the values computed from the signed encodings are too.
const txRollupCostDataTestKey = "b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291"

func signTestTx(t *testing.T, signer types.Signer, txdata types.TxData) *types.Transaction {
	t.Helper()
	key, err := crypto.HexToECDSA(txRollupCostDataTestKey)
	require.NoError(t, err)
	tx, err := types.SignNewTx(key, signer, txdata)
	require.NoError(t, err)
	return tx
}

// txRollupCostDataVectors covers every transaction type a user can send on an L2, unsigned,
// as fee estimation sees it, and signed. The cost data is that of the whole binary-encoded
// transaction, signature included, not of its calldata alone.
func txRollupCostDataVectors(t *testing.T) []txRollupCostDataVector {
	t.Helper()
	chainID := big.NewInt(10)
	sign := func(signer types.Signer, txdata types.TxData) *types.Transaction {
		return signTestTx(t, signer, txdata)
	}
	latest := types.LatestSignerForChainID(chainID)

	to := common.HexToAddress("095e7baea6a6c7c4c2dfeb977efac326af552d87")
	mixedData := []byte{0x00, 0x01, 0x00, 0xff, 0x00, 0x42, 0x00, 0x00, 0xab}
	legacy := &types.LegacyTx{Nonce: 0, To: &to, Value: big.NewInt(1), Gas: 21000, GasPrice: big.NewInt(1)}
	legacyWithData := &types.LegacyTx{Nonce: 7, To: &to, Value: big.NewInt(0), Gas: 50000, GasPrice: big.NewInt(3), Data: mixedData}
	accessList := &types.AccessListTx{ChainID: chainID, Nonce: 1, To: &to, Value: big.NewInt(2), Gas: 30000, GasPrice: big.NewInt(2), Data: mixedData,
		AccessList: types.AccessList{{Address: to, StorageKeys: []common.Hash{{0x02}}}}}
	dynamicFee := &types.DynamicFeeTx{ChainID: chainID, Nonce: 3, GasTipCap: big.NewInt(2), GasFeeCap: big.NewInt(5), Gas: 60000, To: &to, Value: big.NewInt(7), Data: mixedData}
	setCode := &types.SetCodeTx{ChainID: uint256.NewInt(10), Nonce: 4, GasTipCap: uint256.NewInt(2), GasFeeCap: uint256.NewInt(5), Gas: 60000, To: to, Value: uint256.NewInt(0), Data: mixedData,
		AuthList: []types.SetCodeAuthorization{{ChainID: *uint256.NewInt(10), Address: to, Nonce: 5}}}

	return []txRollupCostDataVector{
		{name: "legacy without data", tx: types.NewTx(legacy), want: RollupCostData{Zeroes: 0, Ones: 32, FastLzSize: 33}},
		{name: "legacy with data", tx: types.NewTx(legacyWithData), want: RollupCostData{Zeroes: 5, Ones: 36, FastLzSize: 43}},
		{name: "access list", tx: types.NewTx(accessList), want: RollupCostData{Zeroes: 36, Ones: 66, FastLzSize: 63}},
		{name: "dynamic fee", tx: types.NewTx(dynamicFee), want: RollupCostData{Zeroes: 5, Ones: 40, FastLzSize: 47}},
		{name: "set code", tx: types.NewTx(setCode), want: RollupCostData{Zeroes: 5, Ones: 69, FastLzSize: 59}},
		{name: "signed legacy without EIP-155", tx: sign(types.HomesteadSigner{}, legacyWithData), want: RollupCostData{Zeroes: 5, Ones: 101, FastLzSize: 110}},
		{name: "signed EIP-155 legacy", tx: sign(types.NewEIP155Signer(chainID), legacyWithData), want: RollupCostData{Zeroes: 5, Ones: 101, FastLzSize: 110}},
		{name: "signed access list", tx: sign(latest, accessList), want: RollupCostData{Zeroes: 36, Ones: 130, FastLzSize: 127}},
		{name: "signed dynamic fee", tx: sign(latest, dynamicFee), want: RollupCostData{Zeroes: 5, Ones: 105, FastLzSize: 114}},
		{name: "signed set code", tx: sign(latest, setCode), want: RollupCostData{Zeroes: 5, Ones: 133, FastLzSize: 125}},
	}
}

func TestTxRollupCostDataGoldenVectors(t *testing.T) {
	for _, v := range txRollupCostDataVectors(t) {
		t.Run(v.name, func(t *testing.T) {
			require.Equal(t, v.want, TxRollupCostData(v.tx))
		})
	}
}

// TestDAFootprintConstants pins the L1-attributes layout constants of the DA footprint.
func TestDAFootprintConstants(t *testing.T) {
	require.Equal(t, 176, IsthmusL1AttributesLen)
	require.Equal(t, 178, JovianL1AttributesLen)
	require.Equal(t, uint32(0x3db6be2b), jovianL1AttributesSelector)
}

// isthmusL1AttributesSelector is the function selector of Isthmus-style L1 attributes.
const isthmusL1AttributesSelector uint32 = 0x098999be

// isthmusSelectorJovianL1AttributesData is Jovian-length L1-attributes calldata with the
// Isthmus selector.
func isthmusSelectorJovianL1AttributesData() []byte {
	data := jovianL1AttributesData(7)
	binary.BigEndian.PutUint32(data[:4], isthmusL1AttributesSelector)
	return data
}

var extractDAFootprintGasScalarVectors = []struct {
	name    string
	data    []byte
	want    uint16
	wantErr string
}{
	{"nil", nil, 0, "L1 attributes transaction data too short for DA footprint gas scalar: 0"},
	{"empty", []byte{}, 0, "L1 attributes transaction data too short for DA footprint gas scalar: 0"},
	{"isthmus length", isthmusL1AttributesData(), 0, "L1 attributes transaction data too short for DA footprint gas scalar: 176"},
	{"one byte short", make([]byte, JovianL1AttributesLen-1), 0, "L1 attributes transaction data too short for DA footprint gas scalar: 177"},
	{"wrong selector", isthmusSelectorJovianL1AttributesData(), 0, "L1 attributes transaction data does not have Jovian selector"},
	{"zero scalar", jovianL1AttributesData(0), 0, ""},
	{"scalar one", jovianL1AttributesData(1), 1, ""},
	{"typical scalar", jovianL1AttributesData(400), 400, ""},
	{"max scalar", jovianL1AttributesData(math.MaxUint16), math.MaxUint16, ""},
	{"longer than jovian", append(jovianL1AttributesData(400), 0xff, 0xee), 400, ""},
}

func TestExtractDAFootprintGasScalarGoldenVectors(t *testing.T) {
	for _, v := range extractDAFootprintGasScalarVectors {
		t.Run(v.name, func(t *testing.T) {
			got, err := ExtractDAFootprintGasScalar(v.data)
			if v.wantErr != "" {
				require.EqualError(t, err, v.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, v.want, got)
		})
	}
}

type daFootprintVector struct {
	name string
	// txs are a block's binary-encoded transactions.
	txs     []hexutil.Bytes
	want    uint64
	wantErr string
}

// encodeTx returns the binary encoding of tx.
func encodeTx(t *testing.T, tx interface{ MarshalBinary() ([]byte, error) }) hexutil.Bytes {
	t.Helper()
	raw, err := tx.MarshalBinary()
	require.NoError(t, err)
	return raw
}

// l1AttributesDeposit returns the binary encoding of an L1-attributes deposit with the
// calldata data.
func l1AttributesDeposit(t *testing.T, data []byte) hexutil.Bytes {
	t.Helper()
	to := common.HexToAddress("4200000000000000000000000000000000000015")
	return encodeTx(t, &optypes.DepositTx{
		SourceHash: common.Hash{0x01},
		From:       common.HexToAddress("deaddeaddeaddeaddeaddeaddeaddeaddead0001"),
		To:         &to,
		Gas:        1_000_000,
		Data:       data,
	})
}

// encodedUserTxs returns the binary encodings of the signed transactions of
// txRollupCostDataVectors: only a signed transaction can be in a block.
func encodedUserTxs(t *testing.T) []hexutil.Bytes {
	t.Helper()
	var txs []hexutil.Bytes
	for _, v := range txRollupCostDataVectors(t) {
		if _, r, _ := v.tx.RawSignatureValues(); r.Sign() != 0 {
			txs = append(txs, encodeTx(t, v.tx))
		}
	}
	return txs
}

func daFootprintVectors(t *testing.T) []daFootprintVector {
	t.Helper()
	block := func(l1Attributes []byte, txs ...hexutil.Bytes) []hexutil.Bytes {
		return append([]hexutil.Bytes{l1AttributesDeposit(t, l1Attributes)}, txs...)
	}
	userTxs := encodedUserTxs(t)
	secondDeposit := l1AttributesDeposit(t, nil)
	// largeTx is the only user transaction whose DA-size estimate exceeds the minimum.
	var largeData []byte
	for i := range byte(16) {
		largeData = append(largeData, crypto.Keccak256([]byte{i})...)
	}
	to := common.HexToAddress("095e7baea6a6c7c4c2dfeb977efac326af552d87")
	largeTx := encodeTx(t, signTestTx(t, types.LatestSignerForChainID(big.NewInt(10)), &types.DynamicFeeTx{ChainID: big.NewInt(10), Nonce: 6, GasTipCap: big.NewInt(2), GasFeeCap: big.NewInt(5), Gas: 100_000, To: &to, Data: largeData}))

	return []daFootprintVector{
		{name: "no txs", wantErr: "missing deposit transaction"},
		{name: "first tx not deposit", txs: userTxs[:1], wantErr: "missing deposit transaction"},
		{name: "isthmus attributes, deposit only", txs: block(isthmusL1AttributesData())},
		{name: "isthmus attributes, two deposits", txs: block(isthmusL1AttributesData(), secondDeposit)},
		{name: "isthmus attributes, user txs", txs: block(isthmusL1AttributesData(), userTxs...), wantErr: "unexpected non-deposit transactions in Jovian activation block"},
		{name: "jovian attributes, deposit only", txs: block(jovianL1AttributesData(400))},
		{name: "jovian attributes, single legacy tx", txs: block(jovianL1AttributesData(400), userTxs[0]), want: 40_000},
		{name: "jovian attributes, mixed user txs", txs: block(jovianL1AttributesData(400), userTxs...), want: 200_000},
		{name: "jovian attributes, extra deposit and user txs", txs: block(jovianL1AttributesData(400), append([]hexutil.Bytes{secondDeposit}, userTxs...)...), want: 200_000},
		{name: "jovian attributes, zero scalar", txs: block(jovianL1AttributesData(0), userTxs...)},
		{name: "jovian attributes, scalar one", txs: block(jovianL1AttributesData(1), userTxs...), want: 500},
		{name: "jovian attributes, max scalar", txs: block(jovianL1AttributesData(math.MaxUint16), userTxs...), want: 32_767_500},
		{name: "jovian attributes, tx above the minimum size", txs: block(jovianL1AttributesData(400), userTxs[0], largeTx), want: 236_000},
		{name: "jovian attributes, wrong selector", txs: block(isthmusSelectorJovianL1AttributesData(), userTxs...), wantErr: "L1 attributes transaction data does not have Jovian selector"},
		{name: "jovian attributes, truncated", txs: block(make([]byte, 100), userTxs...), wantErr: "L1 attributes transaction data too short for DA footprint gas scalar: 100"},
	}
}

func TestCalcDAFootprintGoldenVectors(t *testing.T) {
	for _, v := range daFootprintVectors(t) {
		t.Run(v.name, func(t *testing.T) {
			got, err := CalcDAFootprint(v.txs)
			if v.wantErr != "" {
				require.EqualError(t, err, v.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, v.want, got)
		})
	}
}
