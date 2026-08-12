package geth

import (
	"fmt"

	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rlp"
)

// EnsureAmsterdamBlockAccessList supplies the canonical empty block access list when an
// engine omits it from an Amsterdam payload. The bundled op-geth builds empty access lists
// but does not propagate them through BlockToExecutableData.
func EnsureAmsterdamBlockAccessList(payload *engine.ExecutableData) {
	if payload.BlockAccessList == nil {
		emptyBlockAccessList := hexutil.Bytes(rlp.EmptyList)
		payload.BlockAccessList = &emptyBlockAccessList
	}
}

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
