package operatorfee

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper function to create a BalanceSnapshot with specified values
func newTestSnapshot(block, baseFee, l1Fee, seqFee, opFee, from *big.Int) *BalanceSnapshot {
	return &BalanceSnapshot{
		BlockNumber:         block,
		BaseFeeVaultBalance: baseFee,
		L1FeeVaultBalance:   l1Fee,
		SequencerFeeVault:   seqFee,
		OperatorFeeVault:    opFee,
		FromBalance:         from,
	}
}

func TestBalanceSnapshot_String(t *testing.T) {
	t.Run("NilSnapshot", func(t *testing.T) {
		var bs *BalanceSnapshot
		assert.Equal(t, "nil", bs.String())
	})

	t.Run("ZeroValues", func(t *testing.T) {
		bs := newTestSnapshot(
			big.NewInt(0),
			big.NewInt(0),
			big.NewInt(0),
			big.NewInt(0),
			big.NewInt(0),
			big.NewInt(0),
		)
		expected := "BalanceSnapshot{Block: 0, BaseFeeVault: 0, L1FeeVault: 0, SequencerFeeVault: 0, OperatorFeeVault: 0, WalletBalance: 0}"
		assert.Equal(t, expected, bs.String())
	})

	t.Run("NonZeroValues", func(t *testing.T) {
		bs := newTestSnapshot(
			big.NewInt(100),
			big.NewInt(10),
			big.NewInt(20),
			big.NewInt(30),
			big.NewInt(40),
			big.NewInt(50),
		)
		expected := "BalanceSnapshot{Block: 100, BaseFeeVault: 10, L1FeeVault: 20, SequencerFeeVault: 30, OperatorFeeVault: 40, WalletBalance: 50}"
		assert.Equal(t, expected, bs.String())
	})
}

func TestBalanceSnapshot_Add(t *testing.T) {
	start := newTestSnapshot(
		big.NewInt(100),
		big.NewInt(10),
		big.NewInt(20),
		big.NewInt(30),
		big.NewInt(40),
		big.NewInt(500),
	)
	delta := newTestSnapshot(
		big.NewInt(101), // Block number should come from delta
		big.NewInt(5),
		big.NewInt(10),
		big.NewInt(15),
		big.NewInt(20),
		big.NewInt(100),
	)
	expected := newTestSnapshot(
		big.NewInt(101), // Expected block is from delta
		big.NewInt(15),
		big.NewInt(30),
		big.NewInt(45),
		big.NewInt(60),
		big.NewInt(600),
	)

	t.Run("AddNonNil", func(t *testing.T) {
		result := delta.Add(start)
		require.NotNil(t, result)
		// Direct comparison instead of AssertSnapshotsEqual
		assert.True(t, expected.BlockNumber.Cmp(result.BlockNumber) == 0, "BlockNumber mismatch: expected %v, got %v", expected.BlockNumber, result.BlockNumber)
		assert.True(t, expected.BaseFeeVaultBalance.Cmp(result.BaseFeeVaultBalance) == 0, "BaseFeeVaultBalance mismatch: expected %v, got %v", expected.BaseFeeVaultBalance, result.BaseFeeVaultBalance)
		assert.True(t, expected.L1FeeVaultBalance.Cmp(result.L1FeeVaultBalance) == 0, "L1FeeVaultBalance mismatch: expected %v, got %v", expected.L1FeeVaultBalance, result.L1FeeVaultBalance)
		assert.True(t, expected.SequencerFeeVault.Cmp(result.SequencerFeeVault) == 0, "SequencerFeeVault mismatch: expected %v, got %v", expected.SequencerFeeVault, result.SequencerFeeVault)
		assert.True(t, expected.OperatorFeeVault.Cmp(result.OperatorFeeVault) == 0, "OperatorFeeVault mismatch: expected %v, got %v", expected.OperatorFeeVault, result.OperatorFeeVault)
		assert.True(t, expected.FromBalance.Cmp(result.FromBalance) == 0, "FromBalance mismatch: expected %v, got %v", expected.FromBalance, result.FromBalance)
	})

	t.Run("AddNilStart", func(t *testing.T) {
		result := delta.Add(nil)
		assert.Nil(t, result)
	})

	t.Run("AddNilDelta", func(t *testing.T) {
		var nilDelta *BalanceSnapshot
		result := nilDelta.Add(start)
		assert.Nil(t, result)
	})

	t.Run("AddNilToNil", func(t *testing.T) {
		var nilDelta *BalanceSnapshot
		result := nilDelta.Add(nil)
		assert.Nil(t, result)
	})
}

func TestBalanceSnapshot_Sub(t *testing.T) {
	start := newTestSnapshot(
		big.NewInt(100),
		big.NewInt(10),
		big.NewInt(20),
		big.NewInt(30),
		big.NewInt(40),
		big.NewInt(500),
	)
	end := newTestSnapshot(
		big.NewInt(101), // Block number should come from 'end' (bs)
		big.NewInt(15),
		big.NewInt(30),
		big.NewInt(45),
		big.NewInt(60),
		big.NewInt(600),
	)
	expectedDelta := newTestSnapshot(
		big.NewInt(101), // Expected block is from end (bs)
		big.NewInt(5),
		big.NewInt(10),
		big.NewInt(15),
		big.NewInt(20),
		big.NewInt(100),
	)

	t.Run("SubNonNil", func(t *testing.T) {
		delta := end.Sub(start)
		require.NotNil(t, delta)
		// Direct comparison
		assert.True(t, expectedDelta.BlockNumber.Cmp(delta.BlockNumber) == 0, "BlockNumber mismatch: expected %v, got %v", expectedDelta.BlockNumber, delta.BlockNumber)
		assert.True(t, expectedDelta.BaseFeeVaultBalance.Cmp(delta.BaseFeeVaultBalance) == 0, "BaseFeeVaultBalance mismatch: expected %v, got %v", expectedDelta.BaseFeeVaultBalance, delta.BaseFeeVaultBalance)
		assert.True(t, expectedDelta.L1FeeVaultBalance.Cmp(delta.L1FeeVaultBalance) == 0, "L1FeeVaultBalance mismatch: expected %v, got %v", expectedDelta.L1FeeVaultBalance, delta.L1FeeVaultBalance)
		assert.True(t, expectedDelta.SequencerFeeVault.Cmp(delta.SequencerFeeVault) == 0, "SequencerFeeVault mismatch: expected %v, got %v", expectedDelta.SequencerFeeVault, delta.SequencerFeeVault)
		assert.True(t, expectedDelta.OperatorFeeVault.Cmp(delta.OperatorFeeVault) == 0, "OperatorFeeVault mismatch: expected %v, got %v", expectedDelta.OperatorFeeVault, delta.OperatorFeeVault)
		assert.True(t, expectedDelta.FromBalance.Cmp(delta.FromBalance) == 0, "FromBalance mismatch: expected %v, got %v", expectedDelta.FromBalance, delta.FromBalance)
	})

	t.Run("SubNilStart", func(t *testing.T) {
		delta := end.Sub(nil)
		assert.Nil(t, delta)
	})

	t.Run("SubNilEnd", func(t *testing.T) {
		var nilEnd *BalanceSnapshot
		delta := nilEnd.Sub(start)
		assert.Nil(t, delta)
	})

	t.Run("SubNilFromNil", func(t *testing.T) {
		var nilEnd *BalanceSnapshot
		delta := nilEnd.Sub(nil)
		assert.Nil(t, delta)
	})

	t.Run("SubNegativeResult", func(t *testing.T) {
		// Swapping start and end should result in negative delta
		expectedNegativeDelta := newTestSnapshot(
			big.NewInt(100), // Block number from start (now acting as 'bs')
			big.NewInt(-5),
			big.NewInt(-10),
			big.NewInt(-15),
			big.NewInt(-20),
			big.NewInt(-100),
		)
		delta := start.Sub(end)
		require.NotNil(t, delta)
		// Direct comparison
		assert.True(t, expectedNegativeDelta.BlockNumber.Cmp(delta.BlockNumber) == 0, "BlockNumber mismatch: expected %v, got %v", expectedNegativeDelta.BlockNumber, delta.BlockNumber)
		assert.True(t, expectedNegativeDelta.BaseFeeVaultBalance.Cmp(delta.BaseFeeVaultBalance) == 0, "BaseFeeVaultBalance mismatch: expected %v, got %v", expectedNegativeDelta.BaseFeeVaultBalance, delta.BaseFeeVaultBalance)
		assert.True(t, expectedNegativeDelta.L1FeeVaultBalance.Cmp(delta.L1FeeVaultBalance) == 0, "L1FeeVaultBalance mismatch: expected %v, got %v", expectedNegativeDelta.L1FeeVaultBalance, delta.L1FeeVaultBalance)
		assert.True(t, expectedNegativeDelta.SequencerFeeVault.Cmp(delta.SequencerFeeVault) == 0, "SequencerFeeVault mismatch: expected %v, got %v", expectedNegativeDelta.SequencerFeeVault, delta.SequencerFeeVault)
		assert.True(t, expectedNegativeDelta.OperatorFeeVault.Cmp(delta.OperatorFeeVault) == 0, "OperatorFeeVault mismatch: expected %v, got %v", expectedNegativeDelta.OperatorFeeVault, delta.OperatorFeeVault)
		assert.True(t, expectedNegativeDelta.FromBalance.Cmp(delta.FromBalance) == 0, "FromBalance mismatch: expected %v, got %v", expectedNegativeDelta.FromBalance, delta.FromBalance)
	})
}

func TestSnapshotsEqual(t *testing.T) {
	snap1 := newTestSnapshot(big.NewInt(1), big.NewInt(10), big.NewInt(20), big.NewInt(30), big.NewInt(40), big.NewInt(50))
	snap2 := newTestSnapshot(big.NewInt(1), big.NewInt(10), big.NewInt(20), big.NewInt(30), big.NewInt(40), big.NewInt(50))

	t.Run("EqualSnapshots", func(t *testing.T) {
		assert.True(t, SnapshotsEqual(snap1, snap2), "Equal snapshots should return true")
	})

	t.Run("DifferentBaseFee", func(t *testing.T) {
		diffSnap := newTestSnapshot(big.NewInt(1), big.NewInt(99), big.NewInt(20), big.NewInt(30), big.NewInt(40), big.NewInt(50))
		assert.False(t, SnapshotsEqual(snap1, diffSnap), "Different BaseFeeVaultBalance should return false")
	})

	t.Run("DifferentL1Fee", func(t *testing.T) {
		diffSnap := newTestSnapshot(big.NewInt(1), big.NewInt(10), big.NewInt(99), big.NewInt(30), big.NewInt(40), big.NewInt(50))
		assert.False(t, SnapshotsEqual(snap1, diffSnap), "Different L1FeeVaultBalance should return false")
	})

	t.Run("DifferentSequencerFee", func(t *testing.T) {
		diffSnap := newTestSnapshot(big.NewInt(1), big.NewInt(10), big.NewInt(20), big.NewInt(99), big.NewInt(40), big.NewInt(50))
		assert.False(t, SnapshotsEqual(snap1, diffSnap), "Different SequencerFeeVault should return false")
	})

	t.Run("DifferentOperatorFee", func(t *testing.T) {
		diffSnap := newTestSnapshot(big.NewInt(1), big.NewInt(10), big.NewInt(20), big.NewInt(30), big.NewInt(99), big.NewInt(50))
		assert.False(t, SnapshotsEqual(snap1, diffSnap), "Different OperatorFeeVault should return false")
	})

	t.Run("DifferentFromBalance", func(t *testing.T) {
		diffSnap := newTestSnapshot(big.NewInt(1), big.NewInt(10), big.NewInt(20), big.NewInt(30), big.NewInt(40), big.NewInt(99))
		assert.False(t, SnapshotsEqual(snap1, diffSnap), "Different FromBalance should return false")
	})

	t.Run("NilSnapshots", func(t *testing.T) {
		assert.True(t, SnapshotsEqual(nil, nil), "Both nil should return true")
		assert.False(t, SnapshotsEqual(snap1, nil), "One nil should return false")
		assert.False(t, SnapshotsEqual(nil, snap1), "One nil should return false")
	})
}
