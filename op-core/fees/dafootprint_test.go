package fees

import (
	"encoding/binary"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
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
