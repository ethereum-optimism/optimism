package operatorfee

import (
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/devnet-sdk/testing/systest"
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

// mockTB is a minimal testing.TB implementation for checking assertion failures
// without failing the actual test.
type mockTB struct {
	testing.TB // Embed standard testing.TB for most methods (like Logf)
	failed     bool
}

func (m *mockTB) Helper()                         { m.TB.Helper() }
func (m *mockTB) Errorf(string, ...any)           { m.failed = true }                        // Just record failure
func (m *mockTB) Fatalf(string, ...any)           { m.failed = true; panic("mock Fatalf") }  // Record failure and panic
func (m *mockTB) FailNow()                        { m.failed = true; panic("mock FailNow") } // Record failure and panic
func (m *mockTB) Fail()                           { m.failed = true }                        // Just record failure
func (m *mockTB) Name() string                    { return m.TB.Name() }
func (m *mockTB) Logf(format string, args ...any) { m.TB.Logf(format, args...) }

// Add other testing.TB methods if needed by systest.NewT or AssertSnapshotsEqual
func (m *mockTB) Cleanup(f func())                 { m.TB.Cleanup(f) }
func (m *mockTB) Error(args ...any)                { m.failed = true }
func (m *mockTB) Failed() bool                     { return m.failed } // Reflect our recorded state
func (m *mockTB) Fatal(args ...any)                { m.failed = true; panic("mock Fatal") }
func (m *mockTB) Log(args ...any)                  { m.TB.Log(args...) }
func (m *mockTB) Setenv(key, value string)         { m.TB.Setenv(key, value) }
func (m *mockTB) Skip(args ...any)                 { m.TB.Skip(args...) }
func (m *mockTB) SkipNow()                         { m.TB.SkipNow() }
func (m *mockTB) Skipf(format string, args ...any) { m.TB.Skipf(format, args...) }
func (m *mockTB) Skipped() bool                    { return m.TB.Skipped() }
func (m *mockTB) TempDir() string                  { return m.TB.TempDir() }

func TestAssertSnapshotsEqual(t *testing.T) {
	snap1 := newTestSnapshot(big.NewInt(1), big.NewInt(10), big.NewInt(20), big.NewInt(30), big.NewInt(40), big.NewInt(50))
	snap2 := newTestSnapshot(big.NewInt(1), big.NewInt(10), big.NewInt(20), big.NewInt(30), big.NewInt(40), big.NewInt(50))

	t.Run("EqualSnapshots", func(t *testing.T) {
		mockT := &mockTB{TB: t} // Use the mock TB
		systestT := systest.NewT(mockT)
		AssertSnapshotsEqual(systestT, snap1, snap2)
		assert.False(t, mockT.failed, "AssertSnapshotsEqual should not fail for equal snapshots")
	})

	t.Run("DifferentBaseFee", func(t *testing.T) {
		mockT := &mockTB{TB: t}
		systestT := systest.NewT(mockT)
		diffSnap := newTestSnapshot(big.NewInt(1), big.NewInt(99), big.NewInt(20), big.NewInt(30), big.NewInt(40), big.NewInt(50))
		AssertSnapshotsEqual(systestT, snap1, diffSnap)
		assert.True(t, mockT.failed, "AssertSnapshotsEqual should fail for different BaseFeeVaultBalance")
	})

	t.Run("DifferentL1Fee", func(t *testing.T) {
		mockT := &mockTB{TB: t}
		systestT := systest.NewT(mockT)
		diffSnap := newTestSnapshot(big.NewInt(1), big.NewInt(10), big.NewInt(99), big.NewInt(30), big.NewInt(40), big.NewInt(50))
		AssertSnapshotsEqual(systestT, snap1, diffSnap)
		assert.True(t, mockT.failed, "AssertSnapshotsEqual should fail for different L1FeeVaultBalance")
	})

	t.Run("DifferentSequencerFee", func(t *testing.T) {
		mockT := &mockTB{TB: t}
		systestT := systest.NewT(mockT)
		diffSnap := newTestSnapshot(big.NewInt(1), big.NewInt(10), big.NewInt(20), big.NewInt(99), big.NewInt(40), big.NewInt(50))
		AssertSnapshotsEqual(systestT, snap1, diffSnap)
		assert.True(t, mockT.failed, "AssertSnapshotsEqual should fail for different SequencerFeeVault")
	})

	t.Run("DifferentOperatorFee", func(t *testing.T) {
		mockT := &mockTB{TB: t}
		systestT := systest.NewT(mockT)
		diffSnap := newTestSnapshot(big.NewInt(1), big.NewInt(10), big.NewInt(20), big.NewInt(30), big.NewInt(99), big.NewInt(50))
		AssertSnapshotsEqual(systestT, snap1, diffSnap)
		assert.True(t, mockT.failed, "AssertSnapshotsEqual should fail for different OperatorFeeVault")
	})

	t.Run("DifferentFromBalance", func(t *testing.T) {
		mockT := &mockTB{TB: t}
		systestT := systest.NewT(mockT)
		diffSnap := newTestSnapshot(big.NewInt(1), big.NewInt(10), big.NewInt(20), big.NewInt(30), big.NewInt(40), big.NewInt(99))
		AssertSnapshotsEqual(systestT, snap1, diffSnap)
		assert.True(t, mockT.failed, "AssertSnapshotsEqual should fail for different FromBalance")
	})

	// Test require.NotNil checks within AssertSnapshotsEqual (which call FailNow)
	t.Run("NilExpected", func(t *testing.T) {
		mockT := &mockTB{TB: t}
		systestT := systest.NewT(mockT)
		// Use assert.Panics because require.NotNil calls t.FailNow() which our mock makes panic
		assert.Panics(t, func() {
			AssertSnapshotsEqual(systestT, nil, snap2)
		}, "AssertSnapshotsEqual should panic via FailNow when expected is nil")
		assert.True(t, mockT.failed) // Check if FailNow was triggered
	})

	t.Run("NilActual", func(t *testing.T) {
		mockT := &mockTB{TB: t}
		systestT := systest.NewT(mockT)
		assert.Panics(t, func() {
			AssertSnapshotsEqual(systestT, snap1, nil)
		}, "AssertSnapshotsEqual should panic via FailNow when actual is nil")
		assert.True(t, mockT.failed) // Check if FailNow was triggered
	})
}
