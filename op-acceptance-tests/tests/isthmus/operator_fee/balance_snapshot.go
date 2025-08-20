package operatorfee

import (
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type BalanceSnapshot struct {
	BlockNumber         *big.Int
	BaseFeeVaultBalance *big.Int
	L1FeeVaultBalance   *big.Int
	SequencerFeeVault   *big.Int
	OperatorFeeVault    *big.Int
	FromBalance         *big.Int
}

// String returns a formatted string representation of the balance snapshot
func (bs *BalanceSnapshot) String() string {
	if bs == nil {
		return "nil"
	}

	return fmt.Sprintf(
		"BalanceSnapshot{Block: %v, BaseFeeVault: %v, L1FeeVault: %v, SequencerFeeVault: %v, "+
			"OperatorFeeVault: %v, WalletBalance: %v}",
		bs.BlockNumber,
		bs.BaseFeeVaultBalance,
		bs.L1FeeVaultBalance,
		bs.SequencerFeeVault,
		bs.OperatorFeeVault,
		bs.FromBalance,
	)
}

// Add adds this snapshot's balances to another snapshot and returns a new snapshot
// This is typically used to apply changes to a starting balance snapshot
func (bs *BalanceSnapshot) Add(start *BalanceSnapshot) *BalanceSnapshot {
	if bs == nil || start == nil {
		return nil
	}

	return &BalanceSnapshot{
		BlockNumber:         bs.BlockNumber, // Use the target block number from changes
		BaseFeeVaultBalance: new(big.Int).Add(start.BaseFeeVaultBalance, bs.BaseFeeVaultBalance),
		L1FeeVaultBalance:   new(big.Int).Add(start.L1FeeVaultBalance, bs.L1FeeVaultBalance),
		SequencerFeeVault:   new(big.Int).Add(start.SequencerFeeVault, bs.SequencerFeeVault),
		OperatorFeeVault:    new(big.Int).Add(start.OperatorFeeVault, bs.OperatorFeeVault),
		FromBalance:         new(big.Int).Add(start.FromBalance, bs.FromBalance),
	}
}

// Sub returns a new BalanceSnapshot containing the differences between this snapshot and another
// This snapshot is considered the "end" and the parameter is the "start"
// Positive values indicate increases, negative values indicate decreases
func (bs *BalanceSnapshot) Sub(start *BalanceSnapshot) *BalanceSnapshot {
	if bs == nil || start == nil {
		return nil
	}

	return &BalanceSnapshot{
		BlockNumber:         bs.BlockNumber,
		BaseFeeVaultBalance: new(big.Int).Sub(bs.BaseFeeVaultBalance, start.BaseFeeVaultBalance),
		L1FeeVaultBalance:   new(big.Int).Sub(bs.L1FeeVaultBalance, start.L1FeeVaultBalance),
		SequencerFeeVault:   new(big.Int).Sub(bs.SequencerFeeVault, start.SequencerFeeVault),
		OperatorFeeVault:    new(big.Int).Sub(bs.OperatorFeeVault, start.OperatorFeeVault),
		FromBalance:         new(big.Int).Sub(bs.FromBalance, start.FromBalance),
	}
}

// AssertSnapshotsEqual compares two balance snapshots and reports differences
func AssertSnapshotsEqual(t devtest.T, expected, actual *BalanceSnapshot) {
	require.NotNil(t, expected, "Expected snapshot should not be nil")
	require.NotNil(t, actual, "Actual snapshot should not be nil")

	// Check base fee vault balance
	assert.True(t, expected.BaseFeeVaultBalance.Cmp(actual.BaseFeeVaultBalance) == 0,
		"BaseFeeVaultBalance mismatch: expected %v, got %v (diff: %v)", expected.BaseFeeVaultBalance, actual.BaseFeeVaultBalance, new(big.Int).Sub(actual.BaseFeeVaultBalance, expected.BaseFeeVaultBalance))

	// Check L1 fee vault balance
	assert.True(t, expected.L1FeeVaultBalance.Cmp(actual.L1FeeVaultBalance) == 0,
		"L1FeeVaultBalance mismatch: expected %v, got %v (diff: %v)", expected.L1FeeVaultBalance, actual.L1FeeVaultBalance, new(big.Int).Sub(actual.L1FeeVaultBalance, expected.L1FeeVaultBalance))

	// Check sequencer fee vault balance
	assert.True(t, expected.SequencerFeeVault.Cmp(actual.SequencerFeeVault) == 0,
		"SequencerFeeVault mismatch: expected %v, got %v (diff: %v)", expected.SequencerFeeVault, actual.SequencerFeeVault, new(big.Int).Sub(actual.SequencerFeeVault, expected.SequencerFeeVault))

	// Check operator fee vault balance
	assert.True(t, expected.OperatorFeeVault.Cmp(actual.OperatorFeeVault) == 0,
		"OperatorFeeVault mismatch: expected %v, got %v (diff: %v)", expected.OperatorFeeVault, actual.OperatorFeeVault, new(big.Int).Sub(actual.OperatorFeeVault, expected.OperatorFeeVault))

	// Check wallet balance
	assert.True(t, expected.FromBalance.Cmp(actual.FromBalance) == 0,
		"WalletBalance mismatch: expected %v, got %v (diff: %v)", expected.FromBalance, actual.FromBalance, new(big.Int).Sub(actual.FromBalance, expected.FromBalance))
}

// SnapshotsEqual compares two balance snapshots and returns true if they are equal
// This is a non-asserting version for unit tests
func SnapshotsEqual(expected, actual *BalanceSnapshot) bool {
	if expected == nil || actual == nil {
		return expected == actual
	}

	return expected.BaseFeeVaultBalance.Cmp(actual.BaseFeeVaultBalance) == 0 &&
		expected.L1FeeVaultBalance.Cmp(actual.L1FeeVaultBalance) == 0 &&
		expected.SequencerFeeVault.Cmp(actual.SequencerFeeVault) == 0 &&
		expected.OperatorFeeVault.Cmp(actual.OperatorFeeVault) == 0 &&
		expected.FromBalance.Cmp(actual.FromBalance) == 0
}
