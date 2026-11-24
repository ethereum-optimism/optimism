package fees

import (
	"math/big"
	"testing"

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

// TestEcotoneL1CostFunc pins L1CostEcotone to op-geth's TestEcotoneL1CostFunc reference
// (rollup_cost_test.go): baseFee=1000*1e6, blobBaseFee=10*1e6, scalars 2 and 3, and a calldata
// gas of 480 (op-geth's ecotoneGas; here 30 non-zero bytes) yield ecotoneFee == 960900.
func TestEcotoneL1CostFunc(t *testing.T) {
	c := L1CostEcotone(RollupCostData{Ones: 30}, L1FeeParams{
		L1BaseFee:     big.NewInt(1000 * 1e6),
		L1BlobBaseFee: big.NewInt(10 * 1e6),
		BaseFeeScalar: big.NewInt(2),
		BlobFeeScalar: big.NewInt(3),
	})
	require.Equal(t, big.NewInt(960900), c)
}

func TestFjordL1CostFuncMinimumBounds(t *testing.T) {
	params := L1FeeParams{
		L1BaseFee:     fjordBaseFee,
		L1BlobBaseFee: fjordBlobBaseFee,
		BaseFeeScalar: fjordBaseFeeScalar,
		BlobFeeScalar: fjordBlobBaseFeeScalar,
	}

	// FastLZ sizes below the regression's minimum all clamp to the minimum-size fee.
	// -42.5856 + 0.8365*{110,150,170} all stay below the 100-byte floor.
	for _, fastLzSize := range []uint64{100, 150, 170} {
		c := L1CostFjord(RollupCostData{FastLzSize: fastLzSize}, params)
		require.Equal(t, fjordFee, c, "fastLzSize=%d should clamp to the minimum fee", fastLzSize)
	}

	// Larger transactions exceed the minimum and cost strictly more.
	for _, fastLzSize := range []uint64{171, 175, 200} {
		c := L1CostFjord(RollupCostData{FastLzSize: fastLzSize}, params)
		require.Positive(t, c.Cmp(fjordFee), "fastLzSize=%d should exceed the minimum fee", fastLzSize)
	}
}

// TestFjordL1CostSolidityParity pins the Fjord cost function to the same reference output as
// op-geth's Solidity-parity test.
func TestFjordL1CostSolidityParity(t *testing.T) {
	c := L1CostFjord(RollupCostData{FastLzSize: 235}, L1FeeParams{
		L1BaseFee:     big.NewInt(2 * 1e6),
		L1BlobBaseFee: big.NewInt(3 * 1e6),
		BaseFeeScalar: big.NewInt(20),
		BlobFeeScalar: big.NewInt(15),
	})
	require.Equal(t, big.NewInt(105484), c)
}

// TestL1CostFjordFromTxBytes prices transaction bytes from compression to fee, below and above the
// minimum-size floor of the DA-size estimate. The expected fees were taken from the GasPriceOracle
// predeploy: its getL1Fee, executed in an EVM with Fjord active and the L1 fee parameters of
// gweiL1FeeParams, returns them for the same bytes. getL1Fee adds 68 bytes for a signature to the
// FastLZ size of the bytes it is given, so the test does the same. contractCallTx is already
// signed; here it only serves as realistic bytes above the floor.
func TestL1CostFjordFromTxBytes(t *testing.T) {
	testCases := []struct {
		name       string
		tx         []byte
		wantDASize int64
		wantFee    string
	}{
		{"empty tx below the floor", emptyTx, 100, "7701347450000"},
		{"contract call tx above the floor", contractCallTx, 183, "14114213263530"},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rcd := NewRollupCostData(tc.tx)
			rcd.FastLzSize += 68
			require.Equal(t, big.NewInt(tc.wantDASize), rcd.EstimatedDASize())
			require.Equal(t, tc.wantFee, L1CostFjord(rcd, gweiL1FeeParams).String())
		})
	}
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
