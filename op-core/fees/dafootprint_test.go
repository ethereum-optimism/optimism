package fees

import (
	"encoding/binary"
	"math"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	optypes "github.com/ethereum-optimism/optimism/op-core/types"
)

// jovianL1AttributesData builds Jovian-length L1-attributes calldata with the Jovian selector
// and the given daFootprintGasScalar in the trailing two bytes.
func jovianL1AttributesData(scalar uint16) []byte {
	data := make([]byte, JovianL1AttributesLen)
	binary.BigEndian.PutUint32(data[:4], jovianL1AttributesSelector)
	binary.BigEndian.PutUint16(data[JovianL1AttributesLen-2:], scalar)
	return data
}

// isthmusL1AttributesData builds Isthmus-length L1-attributes calldata, as found in the very
// first Jovian block (the activation block still carries Isthmus-style attributes).
func isthmusL1AttributesData() []byte {
	data := make([]byte, IsthmusL1AttributesLen)
	copy(data, types.IsthmusL1AttributesSelector)
	return data
}

func l1AttributesDepositTx(data []byte) *types.Transaction {
	from := common.HexToAddress("deaddeaddeaddeaddeaddeaddeaddeaddead0001")
	to := common.HexToAddress("4200000000000000000000000000000000000015")
	return types.NewTx(&types.DepositTx{
		SourceHash: common.Hash{0x01},
		From:       from,
		To:         &to,
		Gas:        1_000_000,
		Data:       data,
	})
}

// userTxMix returns one transaction of every non-deposit type the fee schedule must cover:
// legacy, access-list, dynamic-fee, SetCode, and blob.
func userTxMix() []*types.Transaction {
	to := common.HexToAddress("095e7baea6a6c7c4c2dfeb977efac326af552d87")
	mixedData := []byte{0x00, 0x01, 0x00, 0xff, 0x00, 0x42, 0x00, 0x00, 0xab}
	return []*types.Transaction{
		types.NewTx(&types.LegacyTx{Nonce: 0, To: &to, Value: big.NewInt(1), Gas: 21000, GasPrice: big.NewInt(1)}),
		types.NewTx(&types.AccessListTx{ChainID: big.NewInt(10), Nonce: 1, To: &to, Value: big.NewInt(2), Gas: 30000, GasPrice: big.NewInt(2), Data: mixedData,
			AccessList: types.AccessList{{Address: to, StorageKeys: []common.Hash{{0x02}}}}}),
		types.NewTx(&types.DynamicFeeTx{ChainID: big.NewInt(10), Nonce: 3, GasTipCap: big.NewInt(2), GasFeeCap: big.NewInt(5), Gas: 60000, To: &to, Value: big.NewInt(7), Data: mixedData}),
		types.NewTx(&types.SetCodeTx{ChainID: uint256.NewInt(10), Nonce: 4, GasTipCap: uint256.NewInt(2), GasFeeCap: uint256.NewInt(5), Gas: 60000, To: to, Value: uint256.NewInt(0), Data: mixedData,
			AuthList: []types.SetCodeAuthorization{{ChainID: *uint256.NewInt(10), Address: to, Nonce: 5}}}),
		types.NewTx(&types.BlobTx{ChainID: uint256.NewInt(10), Nonce: 5, GasTipCap: uint256.NewInt(2), GasFeeCap: uint256.NewInt(5), Gas: 21000, To: to, Value: uint256.NewInt(0), Data: mixedData,
			BlobFeeCap: uint256.NewInt(3), BlobHashes: []common.Hash{{0x01}}}),
	}
}

// TestDAFootprintConstantsParity pins the ported L1-attributes constants to op-geth's.
func TestDAFootprintConstantsParity(t *testing.T) {
	require.Equal(t, types.IsthmusL1AttributesLen, IsthmusL1AttributesLen)
	require.Equal(t, types.JovianL1AttributesLen, JovianL1AttributesLen)
	require.Equal(t, binary.BigEndian.Uint32(types.JovianL1AttributesSelector), jovianL1AttributesSelector)
}

// TestExtractDAFootprintGasScalarParity is a live differential check that
// ExtractDAFootprintGasScalar matches op-geth's types.ExtractDAFootprintGasScalar, including
// error cases. It runs while the build still resolves go-ethereum to op-geth; remove it when
// the op-geth dependency is dropped.
func TestExtractDAFootprintGasScalarParity(t *testing.T) {
	wrongSelector := jovianL1AttributesData(7)
	copy(wrongSelector, types.IsthmusL1AttributesSelector)

	cases := []struct {
		name string
		data []byte
	}{
		{"nil", nil},
		{"empty", []byte{}},
		{"isthmus length", isthmusL1AttributesData()},
		{"one byte short", make([]byte, JovianL1AttributesLen-1)},
		{"wrong selector", wrongSelector},
		{"zero scalar", jovianL1AttributesData(0)},
		{"scalar one", jovianL1AttributesData(1)},
		{"typical scalar", jovianL1AttributesData(400)},
		{"max scalar", jovianL1AttributesData(math.MaxUint16)},
		{"longer than jovian", append(jovianL1AttributesData(400), 0xff, 0xee)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			want, wantErr := types.ExtractDAFootprintGasScalar(tc.data)
			got, gotErr := ExtractDAFootprintGasScalar(tc.data)
			require.Equal(t, want, got)
			if wantErr != nil {
				require.EqualError(t, gotErr, wantErr.Error())
			} else {
				require.NoError(t, gotErr)
			}
		})
	}
}

// TestCalcDAFootprintParity is a live differential check that CalcDAFootprint matches op-geth's
// types.CalcDAFootprint across pre-Lagoon deposit/user-transaction mixes, attribute layouts, and
// scalar values, including error cases. Post-exec transactions intentionally diverge because
// op-geth does not support Lagoon. Remove this test when the op-geth dependency is dropped.
func TestCalcDAFootprintParity(t *testing.T) {
	userTxs := userTxMix()
	secondDeposit := l1AttributesDepositTx(nil)

	wrongSelector := jovianL1AttributesData(7)
	copy(wrongSelector, types.IsthmusL1AttributesSelector)

	cases := []struct {
		name string
		txs  []*types.Transaction
	}{
		{"no txs", nil},
		{"first tx not deposit", []*types.Transaction{userTxs[0]}},
		{"isthmus attributes, deposit only", []*types.Transaction{l1AttributesDepositTx(isthmusL1AttributesData())}},
		{"isthmus attributes, two deposits", []*types.Transaction{l1AttributesDepositTx(isthmusL1AttributesData()), secondDeposit}},
		{"isthmus attributes, user txs", append([]*types.Transaction{l1AttributesDepositTx(isthmusL1AttributesData())}, userTxs...)},
		{"jovian attributes, deposit only", []*types.Transaction{l1AttributesDepositTx(jovianL1AttributesData(400))}},
		{"jovian attributes, mixed user txs", append([]*types.Transaction{l1AttributesDepositTx(jovianL1AttributesData(400))}, userTxs...)},
		{"jovian attributes, extra deposit and user txs", append([]*types.Transaction{l1AttributesDepositTx(jovianL1AttributesData(400)), secondDeposit}, userTxs...)},
		{"jovian attributes, single legacy tx", []*types.Transaction{l1AttributesDepositTx(jovianL1AttributesData(400)), userTxs[0]}},
		{"jovian attributes, zero scalar", append([]*types.Transaction{l1AttributesDepositTx(jovianL1AttributesData(0))}, userTxs...)},
		{"jovian attributes, scalar one", append([]*types.Transaction{l1AttributesDepositTx(jovianL1AttributesData(1))}, userTxs...)},
		{"jovian attributes, max scalar", append([]*types.Transaction{l1AttributesDepositTx(jovianL1AttributesData(math.MaxUint16))}, userTxs...)},
		{"jovian attributes, wrong selector", append([]*types.Transaction{l1AttributesDepositTx(wrongSelector)}, userTxs...)},
		{"jovian attributes, truncated", append([]*types.Transaction{l1AttributesDepositTx(make([]byte, 100))}, userTxs...)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			want, wantErr := types.CalcDAFootprint(tc.txs)
			got, gotErr := CalcDAFootprint(tc.txs)
			require.Equal(t, want, got)
			if wantErr != nil {
				require.EqualError(t, gotErr, wantErr.Error())
			} else {
				require.NoError(t, gotErr)
			}
		})
	}
}

// TestCalcDAFootprintDirected pins CalcDAFootprint to hand-computed reference values.
func TestCalcDAFootprintDirected(t *testing.T) {
	to := common.HexToAddress("095e7baea6a6c7c4c2dfeb977efac326af552d87")
	// Two small legacy transactions whose Fjord DA-size estimates both clamp to the
	// 100-byte minimum, so the footprint is (100 + 100) * scalar.
	smallTx := func(nonce uint64) *types.Transaction {
		return types.NewTx(&types.LegacyTx{Nonce: nonce, To: &to, Value: big.NewInt(1), Gas: 21000, GasPrice: big.NewInt(1)})
	}
	txs := []*types.Transaction{l1AttributesDepositTx(jovianL1AttributesData(3)), smallTx(0), smallTx(1)}
	got, err := CalcDAFootprint(txs)
	require.NoError(t, err)
	require.Equal(t, uint64(600), got)

	// The Jovian activation block carries Isthmus-length attributes and yields a zero footprint.
	got, err = CalcDAFootprint([]*types.Transaction{l1AttributesDepositTx(isthmusL1AttributesData())})
	require.NoError(t, err)
	require.Zero(t, got)
}

func TestCalcDAFootprintExcludesPostExec(t *testing.T) {
	deposit := l1AttributesDepositTx(jovianL1AttributesData(3))
	userTx := userTxMix()[0]

	rawPostExec, err := (&optypes.PostExecTx{Data: []byte{0xc0}}).MarshalBinary()
	require.NoError(t, err)
	postExecTx := new(types.Transaction)
	require.NoError(t, postExecTx.UnmarshalBinary(rawPostExec))
	require.True(t, optypes.IsPostExecTx(postExecTx))

	withoutPostExec, err := CalcDAFootprint([]*types.Transaction{deposit, userTx})
	require.NoError(t, err)
	withPostExec, err := CalcDAFootprint([]*types.Transaction{deposit, userTx, postExecTx})
	require.NoError(t, err)
	require.Equal(t, withoutPostExec, withPostExec)
}
