package geth

import (
	"fmt"

	"github.com/ethereum/go-ethereum/beacon/engine"
)

// ValidatePayloadStatus requires an Engine API call to have fully validated its payload.
func ValidatePayloadStatus(method string, status engine.PayloadStatusV1) error {
	if status.Status == engine.VALID {
		return nil
	}
	if status.ValidationError != nil {
		return fmt.Errorf("%s returned payload status %s: %s", method, status.Status, *status.ValidationError)
	}
	return fmt.Errorf("%s returned payload status %s", method, status.Status)
}
