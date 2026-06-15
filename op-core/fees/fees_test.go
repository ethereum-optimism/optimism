package fees

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

// Fjord parameter set and expected outcomes, mirroring op-geth's rollup_cost_test.go so the
// ported functions are pinned to the same reference values.
var (
	fjordBaseFee           = big.NewInt(1000 * 1e6)
	fjordBlobBaseFee       = big.NewInt(10 * 1e6)
	fjordBaseFeeScalar     = big.NewInt(2)
	fjordBlobBaseFeeScalar = big.NewInt(3)

	// fjordFee is the cost of the minimum-size transaction:
	// 100_000_000 * (2 * 1000 * 1e6 * 16 + 3 * 10 * 1e6) / 1e12
	fjordFee = big.NewInt(3203000)
)

func TestNewRollupCostData(t *testing.T) {
	rcd := NewRollupCostData([]byte{0x00, 0x01, 0x00, 0x02, 0x00})
	require.Equal(t, uint64(3), rcd.Zeroes)
	require.Equal(t, uint64(2), rcd.Ones)
	require.Equal(t, uint64(FlzCompressLen([]byte{0x00, 0x01, 0x00, 0x02, 0x00})), rcd.FastLzSize)

	empty := NewRollupCostData(nil)
	require.Equal(t, RollupCostData{}, empty)
}

// TestL1CostBedrockParity checks L1CostBedrock against op-geth's exported types.L1Cost.
func TestL1CostBedrockParity(t *testing.T) {
	cases := []struct {
		rollupDataGas               uint64
		l1BaseFee, overhead, scalar *big.Int
	}{
		{1618, big.NewInt(1000 * 1e6), big.NewInt(50), big.NewInt(7 * 1e6)},
		{0, big.NewInt(1), big.NewInt(0), big.NewInt(1_000_000)},
		{530, big.NewInt(2300000), big.NewInt(1000), big.NewInt(684000)},
	}
	for _, tc := range cases {
		want := types.L1Cost(tc.rollupDataGas, tc.l1BaseFee, tc.overhead, tc.scalar)
		got := L1CostBedrock(tc.rollupDataGas, tc.l1BaseFee, tc.overhead, tc.scalar)
		require.Zero(t, want.Cmp(got), "want %s, got %s", want, got)
	}
}

// TestEcotoneL1CostFunc pins NewL1CostFuncEcotone to op-geth's TestEcotoneL1CostFunc reference
// (rollup_cost_test.go): baseFee=1000*1e6, blobBaseFee=10*1e6, scalars 2 and 3, and a calldata
// gas of 480 (op-geth's ecotoneGas; here 30 non-zero bytes) yield ecotoneFee == 960900.
func TestEcotoneL1CostFunc(t *testing.T) {
	costFunc := NewL1CostFuncEcotone(big.NewInt(1000*1e6), big.NewInt(10*1e6), big.NewInt(2), big.NewInt(3))
	c := costFunc(RollupCostData{Ones: 30}, 0)
	require.Equal(t, big.NewInt(960900), c)
}

func TestFjordL1CostFuncMinimumBounds(t *testing.T) {
	costFunc := NewL1CostFuncFjord(fjordBaseFee, fjordBlobBaseFee, fjordBaseFeeScalar, fjordBlobBaseFeeScalar)

	// FastLZ sizes below the regression's minimum all clamp to the minimum-size fee.
	// -42.5856 + 0.8365*{110,150,170} all stay below the 100-byte floor.
	for _, fastLzSize := range []uint64{100, 150, 170} {
		c := costFunc(RollupCostData{FastLzSize: fastLzSize}, 0)
		require.Equal(t, fjordFee, c, "fastLzSize=%d should clamp to the minimum fee", fastLzSize)
	}

	// Larger transactions exceed the minimum and cost strictly more.
	for _, fastLzSize := range []uint64{171, 175, 200} {
		c := costFunc(RollupCostData{FastLzSize: fastLzSize}, 0)
		require.Positive(t, c.Cmp(fjordFee), "fastLzSize=%d should exceed the minimum fee", fastLzSize)
	}
}

// TestFjordL1CostSolidityParity pins the Fjord cost function to the same reference output as
// op-geth's Solidity-parity test.
func TestFjordL1CostSolidityParity(t *testing.T) {
	costFunc := NewL1CostFuncFjord(big.NewInt(2*1e6), big.NewInt(3*1e6), big.NewInt(20), big.NewInt(15))
	c := costFunc(RollupCostData{FastLzSize: 235}, 0)
	require.Equal(t, big.NewInt(105484), c)
}

func TestFjordL1CostFuncIgnoresBlockTime(t *testing.T) {
	costFunc := NewL1CostFuncFjord(fjordBaseFee, fjordBlobBaseFee, fjordBaseFeeScalar, fjordBlobBaseFeeScalar)
	rcd := RollupCostData{FastLzSize: 500}
	require.Equal(t, costFunc(rcd, 0), costFunc(rcd, 1_000_000_000))
}

func TestEstimatedDASize(t *testing.T) {
	// Below the regression minimum, the estimate clamps to MinTransactionSize (100).
	small := RollupCostData{FastLzSize: 50}
	require.Equal(t, MinTransactionSize, small.EstimatedDASize())

	// Above the minimum: floor((-42_585_600 + 836_500*200) / 1e6) = floor(124.7144) = 124.
	large := RollupCostData{FastLzSize: 200}
	require.Equal(t, big.NewInt(124), large.EstimatedDASize())
}

// TestOperatorCostIsthmus pins OperatorCostIsthmus to op-geth's reference value:
// 1618 * 1439103868 / 1e6 + 1256417826609331460 == 1256417826611659930.
func TestOperatorCostIsthmus(t *testing.T) {
	expected, ok := new(big.Int).SetString("1256417826611659930", 10)
	require.True(t, ok)
	got := OperatorCostIsthmus(1618, 1439103868, 1256417826609331460)
	require.Equal(t, expected, got)

	// Zero scalar and constant yield zero fee regardless of gas.
	require.Equal(t, 0, OperatorCostIsthmus(21000, 0, 0).Sign())
}

// TestOperatorCostJovian pins OperatorCostJovian to op-geth's reference value:
// 1618 * 1439103868 * 100 + 1256417826609331460 == 1256650673615173860.
func TestOperatorCostJovian(t *testing.T) {
	expected, ok := new(big.Int).SetString("1256650673615173860", 10)
	require.True(t, ok)
	got := OperatorCostJovian(1618, 1439103868, 1256417826609331460)
	require.Equal(t, expected, got)

	// Zero scalar and constant yield zero fee regardless of gas.
	require.Equal(t, 0, OperatorCostJovian(21000, 0, 0).Sign())
}

// TestTxRollupCostDataParity is the decisive check that TxRollupCostData faithfully replaces
// op-geth's (*types.Transaction).RollupCostData(): cost data is derived from the full
// binary-encoded transaction (MarshalBinary), not from tx.Data(), and deposits are empty.
func TestTxRollupCostDataParity(t *testing.T) {
	to := common.HexToAddress("095e7baea6a6c7c4c2dfeb977efac326af552d87")
	mixedData := []byte{0x00, 0x01, 0x00, 0xff, 0x00, 0x42, 0x00, 0x00, 0xab}

	txs := []struct {
		name string
		tx   *types.Transaction
	}{
		{"legacy no data", types.NewTx(&types.LegacyTx{Nonce: 0, To: &to, Value: big.NewInt(1), Gas: 21000, GasPrice: big.NewInt(1)})},
		{"legacy with data", types.NewTx(&types.LegacyTx{Nonce: 7, To: &to, Value: big.NewInt(0), Gas: 50000, GasPrice: big.NewInt(3), Data: mixedData})},
		{"dynamic fee", types.NewTx(&types.DynamicFeeTx{ChainID: big.NewInt(10), Nonce: 3, GasTipCap: big.NewInt(2), GasFeeCap: big.NewInt(5), Gas: 60000, To: &to, Value: big.NewInt(7), Data: mixedData})},
		{"blob", types.NewTx(&types.BlobTx{ChainID: uint256.NewInt(10), Nonce: 1, GasTipCap: uint256.NewInt(2), GasFeeCap: uint256.NewInt(5), Gas: 21000, To: to, Value: uint256.NewInt(0), Data: mixedData, BlobFeeCap: uint256.NewInt(3), BlobHashes: []common.Hash{{0x01}}})},
		{"deposit", types.NewTx(&types.DepositTx{SourceHash: common.Hash{0x01}, From: to, To: &to, Value: big.NewInt(0), Gas: 21000, Data: mixedData})},
	}

	for _, tc := range txs {
		t.Run(tc.name, func(t *testing.T) {
			want := tc.tx.RollupCostData() // op-geth's method, the behaviour we must preserve
			got := TxRollupCostData(tc.tx)
			require.Equal(t, want.Zeroes, got.Zeroes, "Zeroes")
			require.Equal(t, want.Ones, got.Ones, "Ones")
			require.Equal(t, want.FastLzSize, got.FastLzSize, "FastLzSize")
		})
	}
}

func TestTxRollupCostDataDepositIsEmpty(t *testing.T) {
	to := common.HexToAddress("095e7baea6a6c7c4c2dfeb977efac326af552d87")
	deposit := types.NewTx(&types.DepositTx{
		SourceHash: common.Hash{0x01},
		From:       to,
		To:         &to,
		Value:      big.NewInt(0),
		Gas:        21000,
		Data:       []byte{0x01, 0x02, 0x03},
	})
	require.Equal(t, RollupCostData{}, TxRollupCostData(deposit))
}
