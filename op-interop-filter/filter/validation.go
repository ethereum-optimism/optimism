package filter

import (
	"fmt"

	"github.com/ethereum-optimism/optimism/op-service/safemath"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

// ValidateMessageTiming validates timing constraints for cross-chain messages.
// Checks:
//   - initMsgTimestamp < inclusionTimestamp (init must be before execution)
//   - initMsgTimestamp + expiryWindow >= inclusionTimestamp (not expired)
func ValidateMessageTiming(initMsgTimestamp, inclusionTimestamp, expiryWindow uint64) error {
	if !(initMsgTimestamp < inclusionTimestamp) {
		return fmt.Errorf("initiating message timestamp %d not before inclusion timestamp %d: %w",
			initMsgTimestamp, inclusionTimestamp, types.ErrConflict)
	}

	expiresAt := safemath.SaturatingAdd(initMsgTimestamp, expiryWindow)
	if expiresAt < inclusionTimestamp {
		return fmt.Errorf("initiating message expired: init %d + expiry window %d = %d < inclusion %d: %w",
			initMsgTimestamp, expiryWindow, expiresAt, inclusionTimestamp, types.ErrConflict)
	}

	return nil
}
