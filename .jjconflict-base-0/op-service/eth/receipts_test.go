package eth_test

import (
	"fmt"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

// newLog creates a log with a specific index and address.
// The address is derived from the index for uniqueness in tests.
func newLog(index uint) *ethtypes.Log {
	// Use index to generate a unique-ish address for testing
	addrBytes := make([]byte, common.AddressLength)
	addrBytes[common.AddressLength-1] = byte(index % 256)
	addr := common.BytesToAddress(addrBytes)
	return &ethtypes.Log{
		Address: addr,
		Index:   index,
		// Other fields (Topics, Data, BlockNumber, TxHash, etc.) are omitted for simplicity
	}
}

// newReceipt creates a receipt containing the given logs.
func newReceipt(logs ...*ethtypes.Log) *ethtypes.Receipt {
	return &ethtypes.Receipt{
		Logs: logs,
		// Other fields omitted
	}
}

// newEmptyReceipt creates a receipt with no logs.
func newEmptyReceipt() *ethtypes.Receipt {
	return &ethtypes.Receipt{
		Logs: []*ethtypes.Log{},
	}
}

// newContiguousReceiptsAndLogs generates a slice of receipts where log indices are globally contiguous.
// It also returns a map of the generated logs keyed by their index for easy lookup in tests.
// logCountsPerReceipt defines the number of logs in each receipt. Use 0 for an empty receipt.
func newContiguousReceiptsAndLogs(logCountsPerReceipt []int) ([]*ethtypes.Receipt, map[uint]*ethtypes.Log) {
	receipts := make([]*ethtypes.Receipt, 0, len(logCountsPerReceipt))
	allLogs := make(map[uint]*ethtypes.Log)
	currentLogIndex := uint(0)

	for _, count := range logCountsPerReceipt {
		if count == 0 {
			receipts = append(receipts, newEmptyReceipt())
		} else {
			logsInReceipt := make([]*ethtypes.Log, 0, count)
			for i := 0; i < count; i++ {
				log := newLog(currentLogIndex)
				logsInReceipt = append(logsInReceipt, log)
				allLogs[currentLogIndex] = log
				currentLogIndex++
			}
			receipts = append(receipts, newReceipt(logsInReceipt...))
		}
	}
	return receipts, allLogs
}

func TestGetLogAtIndex(t *testing.T) {
	type testCase struct {
		name                string
		logCountsPerReceipt []int  // Defines the structure of receipts and logs
		logIndexToFind      uint   // The log index we are looking for
		expectFound         bool   // Whether we expect to find the log
		expectedError       bool   // Whether an error is expected (implies !expectFound)
		errorContains       string // Substring expected in the error message if expectedError is true
	}

	// Define test cases using logCountsPerReceipt
	testCases := []testCase{
		// --- Basic Cases ---
		{name: "single_receipt_single_log_find_0", logCountsPerReceipt: []int{1}, logIndexToFind: 0, expectFound: true},
		{name: "single_receipt_single_log_find_1_too_high", logCountsPerReceipt: []int{1}, logIndexToFind: 1, expectFound: false, expectedError: true, errorContains: "not found"},
		{name: "single_receipt_multi_log_find_first_0", logCountsPerReceipt: []int{3}, logIndexToFind: 0, expectFound: true},
		{name: "single_receipt_multi_log_find_middle_1", logCountsPerReceipt: []int{3}, logIndexToFind: 1, expectFound: true},
		{name: "single_receipt_multi_log_find_last_2", logCountsPerReceipt: []int{3}, logIndexToFind: 2, expectFound: true},
		{name: "single_receipt_multi_log_find_3_too_high", logCountsPerReceipt: []int{3}, logIndexToFind: 3, expectFound: false, expectedError: true, errorContains: "not found"},

		// --- Multi-Receipt Cases ---
		{name: "multi_receipt_find_first_receipt_first_log_0", logCountsPerReceipt: []int{2, 1}, logIndexToFind: 0, expectFound: true},
		{name: "multi_receipt_find_first_receipt_last_log_1", logCountsPerReceipt: []int{2, 1}, logIndexToFind: 1, expectFound: true},
		{name: "multi_receipt_find_second_receipt_first_log_2", logCountsPerReceipt: []int{2, 1}, logIndexToFind: 2, expectFound: true},
		{name: "multi_receipt_find_3_too_high", logCountsPerReceipt: []int{2, 1}, logIndexToFind: 3, expectFound: false, expectedError: true, errorContains: "not found"},

		// --- Cases with Empty Receipts ---
		{name: "empty_at_start_find_first_actual_0", logCountsPerReceipt: []int{0, 2}, logIndexToFind: 0, expectFound: true},
		{name: "empty_at_start_find_second_actual_2", logCountsPerReceipt: []int{0, 2}, logIndexToFind: 1, expectFound: true}, // FAILING
		{name: "empty_at_start_find_2_too_high", logCountsPerReceipt: []int{0, 2}, logIndexToFind: 2, expectFound: false, expectedError: true, errorContains: "not found"},
		{name: "empty_in_middle_find_first_receipt_log_0", logCountsPerReceipt: []int{1, 0, 2}, logIndexToFind: 0, expectFound: true},
		{name: "empty_in_middle_find_third_receipt_first_log_1", logCountsPerReceipt: []int{1, 0, 2}, logIndexToFind: 1, expectFound: true},
		{name: "empty_in_middle_find_third_receipt_last_log_2", logCountsPerReceipt: []int{1, 0, 2}, logIndexToFind: 2, expectFound: true},
		{name: "empty_in_middle_find_3_too_high", logCountsPerReceipt: []int{1, 0, 2}, logIndexToFind: 3, expectFound: false, expectedError: true, errorContains: "not found"},
		{name: "empty_at_end_find_first_receipt_log_0", logCountsPerReceipt: []int{2, 0}, logIndexToFind: 0, expectFound: true},
		{name: "empty_at_end_find_first_receipt_last_log_1", logCountsPerReceipt: []int{2, 0}, logIndexToFind: 1, expectFound: true},
		{name: "empty_at_end_find_2_too_high", logCountsPerReceipt: []int{2, 0}, logIndexToFind: 2, expectFound: false, expectedError: true, errorContains: "not found"},

		// --- Edge/Error Cases ---
		{name: "empty_receipts_input", logCountsPerReceipt: []int{}, logIndexToFind: 0, expectFound: false, expectedError: true, errorContains: "not found"},
		{name: "all_receipts_empty", logCountsPerReceipt: []int{0, 0, 0}, logIndexToFind: 0, expectFound: false, expectedError: true, errorContains: "not found"},
		// Note: Index too low (e.g., finding -1) isn't applicable as uint cannot be negative.
		// Finding 0 when the first log index is > 0 is covered by the binary search implicitly.
	}

	// Cache generated data to avoid regeneration in parallel tests
	dataCache := make(map[string]struct {
		receipts []*ethtypes.Receipt
		logs     map[uint]*ethtypes.Log
	})
	for _, tc := range testCases {
		key := fmt.Sprintf("%v", tc.logCountsPerReceipt)
		if _, exists := dataCache[key]; !exists {
			receipts, logs := newContiguousReceiptsAndLogs(tc.logCountsPerReceipt)
			dataCache[key] = struct {
				receipts []*ethtypes.Receipt
				logs     map[uint]*ethtypes.Log
			}{receipts: receipts, logs: logs}
		}
	}

	// Run test cases
	for _, tc := range testCases {
		tc := tc // Capture range variable
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel() // Run tests in parallel

			// Retrieve cached data
			key := fmt.Sprintf("%v", tc.logCountsPerReceipt)
			data := dataCache[key]
			receipts := data.receipts
			allLogs := data.logs

			log, err := eth.GetLogAtIndex(receipts, tc.logIndexToFind)

			if tc.expectedError {
				require.Error(t, err, "Expected an error but got none")
				if tc.errorContains != "" {
					require.ErrorContains(t, err, tc.errorContains, "Error message mismatch")
				}
				require.Nil(t, log, "Expected nil log on error")
			} else if tc.expectFound {
				require.NoError(t, err, "Did not expect an error but got one: %v", err)
				require.NotNil(t, log, "Expected a non-nil log")

				// Look up the expected log from the generated map
				expectedLog, found := allLogs[tc.logIndexToFind]
				require.True(t, found, "Test setup error: expected log for index %d not found in generated map for counts %v", tc.logIndexToFind, tc.logCountsPerReceipt)

				// Compare relevant fields (Address and Index are sufficient for this test)
				require.Equal(t, expectedLog.Address, log.Address, "Log Address mismatch")
				require.Equal(t, expectedLog.Index, log.Index, "Log Index mismatch")
			} else {
				// This branch handles the case where we expect !found && !error.
				// GetLogAtIndex should return (nil, error) when not found based on current implementation.
				// If the function signature changes to allow (nil, nil) on not found, this needs adjustment.
				require.Error(t, err, "Expected an error for not found log, but got none")
				require.Nil(t, log, "Expected nil log when not found")
				if tc.errorContains != "" {
					require.ErrorContains(t, err, tc.errorContains, "Error message mismatch for not found case")
				} else {
					// Default check if no specific error message is provided for the 'not found' case
					require.ErrorContains(t, err, "not found", "Expected 'not found' error substring")
				}
			}
		})
	}
}
