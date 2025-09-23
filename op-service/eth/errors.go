package eth

import (
	"errors"
	"strings"

	"github.com/ethereum/go-ethereum"
)

// MaybeAsNotFoundErr checks if the error is an ethereum.NotFound error
// or has an error string that heuristically indicates that it is this error.
// If so, it returns ethereum.NotFound, otherwise it returns the original error.
//
// Note that correct implementations of the execution layer API must return an empty
// result and no error if the block or header is not found. So using this translation
// hardens against wrong implementations of the execution layer API only.
func MaybeAsNotFoundErr(err error) error {
	if errors.Is(err, ethereum.NotFound) || err == nil {
		return err
	}
	if errStr := strings.ToLower(err.Error()); strings.Contains(errStr, "block not found") ||
		strings.Contains(errStr, "header not found") ||
		strings.Contains(errStr, "unknown block") {
		return errors.Join(err, ethereum.NotFound)
	}
	return err
}
