package receipts

import (
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/core/types"
)

// FindLog searches the array of logs (typically retrieved from a receipt) to find one that can be parsed by the
// supplied parser (usually the Parse<EventName> function from generated bindings for a contract).
// e.g. receipts.FindLog(receipt.Logs, optimismPortal.ParseTransactionDeposited)
// Either the first parsable event is returned or an error with the parse failures.
func FindLog[T any](logs []*types.Log, parser func(types.Log) (T, error)) (T, error) {
	var errs error
	for i, l := range logs {
		parsed, err := parser(*l)
		if err == nil {
			return parsed, nil
		}
		errs = errors.Join(errs, fmt.Errorf("parse log %v: %w", i, err))
	}
	var noMatch T
	return noMatch, fmt.Errorf("no matching log found: %w", errs)
}
