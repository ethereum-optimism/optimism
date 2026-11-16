package fees

import (
	"encoding/binary"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
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
	return make([]byte, IsthmusL1AttributesLen)
}

func l1AttributesDeposit(t *testing.T, data []byte) hexutil.Bytes {
	to := common.HexToAddress("4200000000000000000000000000000000000015")
	raw, err := (&optypes.DepositTx{
		SourceHash: common.Hash{0x01},
		From:       common.HexToAddress("deaddeaddeaddeaddeaddeaddeaddeaddead0001"),
		To:         &to,
		Gas:        1_000_000,
		Data:       data,
	}).MarshalBinary()
	require.NoError(t, err)
	return raw
}

func legacyTx(t *testing.T, nonce uint64) hexutil.Bytes {
	to := common.HexToAddress("095e7baea6a6c7c4c2dfeb977efac326af552d87")
	raw, err := types.NewTx(&types.LegacyTx{Nonce: nonce, To: &to, Value: big.NewInt(1), Gas: 21000, GasPrice: big.NewInt(1)}).MarshalBinary()
	require.NoError(t, err)
	return raw
}

// TestCalcDAFootprintDirected pins CalcDAFootprint to hand-computed reference values.
func TestCalcDAFootprintDirected(t *testing.T) {
	// Two small legacy transactions whose Fjord DA-size estimates both clamp to the
	// 100-byte minimum, so the footprint is (100 + 100) * scalar.
	txs := []hexutil.Bytes{l1AttributesDeposit(t, jovianL1AttributesData(3)), legacyTx(t, 0), legacyTx(t, 1)}
	got, err := CalcDAFootprint(txs)
	require.NoError(t, err)
	require.Equal(t, uint64(600), got)

	// The Jovian activation block carries Isthmus-length attributes and yields a zero footprint.
	got, err = CalcDAFootprint([]hexutil.Bytes{l1AttributesDeposit(t, isthmusL1AttributesData())})
	require.NoError(t, err)
	require.Zero(t, got)
}

func TestCalcDAFootprintExcludesPostExec(t *testing.T) {
	deposit := l1AttributesDeposit(t, jovianL1AttributesData(3))
	userTx := legacyTx(t, 0)
	postExecTx, err := (&optypes.PostExecTx{Data: []byte{0xc0}}).MarshalBinary()
	require.NoError(t, err)

	withoutPostExec, err := CalcDAFootprint([]hexutil.Bytes{deposit, userTx})
	require.NoError(t, err)
	withPostExec, err := CalcDAFootprint([]hexutil.Bytes{deposit, userTx, postExecTx})
	require.NoError(t, err)
	require.Equal(t, withoutPostExec, withPostExec)
}
