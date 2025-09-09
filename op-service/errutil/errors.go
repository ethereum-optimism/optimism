package errutil

import (
	"errors"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum"
)

type errWithData interface {
	ErrorData() interface{}
}

// TryAddRevertReason attempts to extract the revert reason from geth RPC client errors and adds it to the error message.
// This is most useful when attempting to execute gas, as if the transaction reverts this will then show the reason.
func TryAddRevertReason(err error) error {
	var errData errWithData
	ok := errors.As(err, &errData)
	if ok {
		return fmt.Errorf("%w, reason: %v", err, errData.ErrorData())
	} else {
		return err
	}
}

// IsEthereumNotFound determines if an error is likely to be ethereum.NotFound even if it has been serialized and
// recreated through an RPC server (thus losing the specific typing). Since this depends on string matching, it may
// return false positives.
func IsEthereumNotFound(err error) bool {
	// The RPC server will convert the returned error to a string so we can't match on an error type here
	return err != nil && (errors.Is(err, ethereum.NotFound) || strings.Contains(err.Error(), ethereum.NotFound.Error()))
}
