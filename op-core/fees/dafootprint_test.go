package fees

import (
	"encoding/binary"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
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
	binary.BigEndian.PutUint32(data[:4], isthmusL1AttributesSelector)
	return data
}

// TestDAFootprintConstantsParity pins the ported L1-attributes constants to op-geth's.
func TestDAFootprintConstantsParity(t *testing.T) {
	require.Equal(t, types.IsthmusL1AttributesLen, IsthmusL1AttributesLen)
	require.Equal(t, types.JovianL1AttributesLen, JovianL1AttributesLen)
	require.Equal(t, binary.BigEndian.Uint32(types.JovianL1AttributesSelector), jovianL1AttributesSelector)
	require.Equal(t, binary.BigEndian.Uint32(types.IsthmusL1AttributesSelector), isthmusL1AttributesSelector)
}

// TestExtractDAFootprintGasScalarParity is a live differential check that
// ExtractDAFootprintGasScalar matches op-geth's types.ExtractDAFootprintGasScalar, including
// error cases. It runs while the build still resolves go-ethereum to op-geth; remove it when
// the op-geth dependency is dropped.
func TestExtractDAFootprintGasScalarParity(t *testing.T) {
	for _, tc := range extractDAFootprintGasScalarVectors {
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

// decodeTxs decodes binary-encoded transactions into op-geth transactions, which can hold
// deposits.
func decodeTxs(t *testing.T, raw []hexutil.Bytes) []*types.Transaction {
	t.Helper()
	txs := make([]*types.Transaction, len(raw))
	for i, r := range raw {
		txs[i] = new(types.Transaction)
		require.NoError(t, txs[i].UnmarshalBinary(r))
	}
	return txs
}

// TestCalcDAFootprintParity is a live differential check that CalcDAFootprint, given the
// binary encodings of a block's transactions, matches op-geth's types.CalcDAFootprint on the
// transactions themselves, across pre-Lagoon deposit/user-transaction mixes, attribute layouts,
// and scalar values. Post-exec transactions intentionally diverge because op-geth does not
// support Lagoon. Remove this test when the op-geth dependency is dropped.
func TestCalcDAFootprintParity(t *testing.T) {
	for _, tc := range daFootprintVectors(t) {
		t.Run(tc.name, func(t *testing.T) {
			want, wantErr := types.CalcDAFootprint(decodeTxs(t, tc.txs))
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
	// A small legacy and a small dynamic-fee transaction whose Fjord DA-size estimates both clamp
	// to the 100-byte minimum, so the footprint is (100 + 100) * scalar.
	userTxs := encodedUserTxs(t)
	txs := []hexutil.Bytes{l1AttributesDeposit(t, jovianL1AttributesData(3)), userTxs[0], userTxs[3]}
	got, err := CalcDAFootprint(txs)
	require.NoError(t, err)
	require.Equal(t, uint64(600), got)

	// The Jovian activation block carries Isthmus-length attributes and yields a zero footprint.
	got, err = CalcDAFootprint([]hexutil.Bytes{l1AttributesDeposit(t, isthmusL1AttributesData())})
	require.NoError(t, err)
	require.Zero(t, got)
}

func TestCalcDAFootprintExcludesPostExec(t *testing.T) {
	txs := []hexutil.Bytes{l1AttributesDeposit(t, jovianL1AttributesData(3)), encodedUserTxs(t)[0]}
	postExecTx, err := (&optypes.PostExecTx{Data: []byte{0xc0}}).MarshalBinary()
	require.NoError(t, err)

	withoutPostExec, err := CalcDAFootprint(txs)
	require.NoError(t, err)
	withPostExec, err := CalcDAFootprint(append(txs, postExecTx))
	require.NoError(t, err)
	require.Equal(t, withoutPostExec, withPostExec)
}

func TestCalcDAFootprintRejectsEmptyTx(t *testing.T) {
	txs := []hexutil.Bytes{l1AttributesDeposit(t, jovianL1AttributesData(3)), encodedUserTxs(t)[0]}
	_, err := CalcDAFootprint(append(txs, hexutil.Bytes{}))
	require.EqualError(t, err, "empty transaction at index 2")
}
