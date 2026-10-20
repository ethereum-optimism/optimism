package errutil

import (
	"errors"
	"fmt"
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
