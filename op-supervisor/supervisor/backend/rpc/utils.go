package rpc

import (
	"errors"
	"fmt"

	ethtypes "github.com/ethereum/go-ethereum/core/types"
)

var ErrNoComparisonResult = errors.New("no comparison result")

// findComparison attempts to compare target with x[h] using cmp.
// If cmp(x[h], ...) returns ErrNoComparisonResult, it uses the nearest
// comparable neighbours (if any) to infer the correct binary search direction for h.
// Returns:
//   - res < 0: target is likely > h (search right)
//   - res > 0: target is likely < h (search left)
//   - res = 0: target == h (only possible if cmp(x[h], target) returned 0)
//   - err = ErrNoComparisonResult: cannot determine direction relative to h.
//   - err = other: an unexpected error occurred during comparison.
func findComparison[S ~[]E, E, T any](x S, target T, h int, cmp func(E, T, int) (int, error)) (int, error) {
	res, err := cmp(x[h], target, h)
	if !errors.Is(err, ErrNoComparisonResult) {
		// Comparison succeeded directly or failed unexpectedly
		return res, err
	}
	// err is ErrNoComparisonResult because x[h] is empty/incomparable

	// Search outwards alternating left and right for the first comparable neighbour
	n := len(x)
	for diff := 1; ; diff++ {
		foundAnyDirection := false

		// Check Left
		idxLeft := h - diff
		if idxLeft >= 0 {
			foundAnyDirection = true
			resLeft, errLeft := cmp(x[idxLeft], target, idxLeft)
			if !errors.Is(errLeft, ErrNoComparisonResult) {
				if errLeft != nil {
					return 0, errLeft // Unexpected error
				}
				// Found comparable left neighbour x[idxLeft]
				if resLeft == 0 { // target == x[idxLeft]
					return 1, nil // Search left
				} else if resLeft < 0 { // target > x[idxLeft].end
					return -1, nil // Search right
				} else { // resLeft > 0 (target < x[idxLeft].start)
					return 1, nil // Search left
				}
			}
		}

		// Check Right
		idxRight := h + diff
		if idxRight < n {
			foundAnyDirection = true
			resRight, errRight := cmp(x[idxRight], target, idxRight)
			if !errors.Is(errRight, ErrNoComparisonResult) {
				if errRight != nil {
					return 0, errRight // Unexpected error
				}
				// Found comparable right neighbour x[idxRight]
				if resRight == 0 { // target == x[idxRight]
					return -1, nil // Search right
				} else if resRight < 0 { // target > x[idxRight].end
					return -1, nil // Search right
				} else { // resRight > 0 (target < x[idxRight].start)
					return 1, nil // Search left
				}
			}
		}

		if !foundAnyDirection {
			// Both left and right checks went out of bounds in the same iteration
			break
		}
	}

	// Loop finished without finding any comparable neighbour.
	return 0, ErrNoComparisonResult
}

// binarySearchFunc works like slices.binarySearchFunc but handles ErrNoComparisonResult from cmp.
// If cmp returns ErrNoComparisonResult for an element, binarySearchFunc uses findComparison
// to potentially infer the search direction from neighbours.
// This version also assumes monotonic indices.
func binarySearchFunc[S ~[]E, E, T any](x S, target T, cmp func(E, T, int) (int, error)) (int, bool) {
	n := len(x)
	i, j := 0, n
	for i < j {
		h := int(uint(i+j) >> 1) // avoid overflow when computing h
		res, err := findComparison(x, target, h, cmp)

		if err != nil {
			// If findComparison couldn't determine direction OR encountered an unexpected error.
			// Cannot place h relative to target. Search cannot continue reliably.
			return 0, false
		}

		// err is nil, comparison successful (directly or via neighbours)
		if res == 0 {
			// findComparison guarantees res=0 only if cmp(x[h], target) == 0
			return h, true
		}
		if res < 0 { // target > h
			i = h + 1
		} else { // target < h
			j = h
		}
	}
	// Loop terminates when i == j. The potential match is at index i.
	// Based on loop invariants, we expect cmp(x[i-1], target) < 0 and cmp(x[i], target) >= 0

	if i < n {
		// Final check: Does x[i] actually match the target?
		res, err := cmp(x[i], target, i)
		// If error (incl. ErrNoComparisonResult), or res != 0, then it's not a match.
		if err != nil || res != 0 {
			return 0, false
		}
		// err == nil and res == 0: Match found at index i.
		return i, true
	}

	// i == n: Target is larger than all elements, or array was empty.
	return 0, false
}

func GetLogAtIndex(receipts []*ethtypes.Receipt, logIndex uint) (*ethtypes.Log, error) {
	// Find the receipt that might contain our log
	receiptIndex, found := binarySearchFunc(receipts, logIndex, func(receipt *ethtypes.Receipt, target uint, index int) (int, error) {
		if receipt == nil || len(receipt.Logs) == 0 {
			return 0, ErrNoComparisonResult
		}

		firstLogIdx := receipt.Logs[0].Index
		lastLogIdx := receipt.Logs[len(receipt.Logs)-1].Index

		if target < firstLogIdx {
			return 1, nil // Target is smaller than the range
		}
		if target > lastLogIdx {
			return -1, nil // Target is larger than the range
		}
		return 0, nil // Target is within the range (inclusive)
	})

	if !found {
		return nil, fmt.Errorf("Log index %d not found in block", logIndex)
	}

	// We found the correct receipt, now find the log within that receipt
	receipt := receipts[receiptIndex]
	if receipt == nil || len(receipt.Logs) == 0 {
		// This should be impossible if BinarySearchFunc found=true, due to the final check
		return nil, fmt.Errorf("internal error: found receipt index %d but receipt is empty/nil", receiptIndex)
	}

	// Calculate the index relative to the start of the logs in this receipt
	relativeIndex := logIndex - receipt.Logs[0].Index
	if relativeIndex >= uint(len(receipt.Logs)) {
		// This should also be impossible if the cmp function is correct
		return nil, fmt.Errorf("internal error: log index %d out of bounds for receipt %d (len %d, start %d)", logIndex, receiptIndex, len(receipt.Logs), receipt.Logs[0].Index)
	}
	return receipt.Logs[relativeIndex], nil
}
