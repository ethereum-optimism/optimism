package filter

import (
	"context"
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

// QueryFrontend handles supervisor query RPC methods
type QueryFrontend struct {
	backend *Backend
}

// CheckAccessList validates interop executing messages
func (f *QueryFrontend) CheckAccessList(ctx context.Context, inboxEntries []common.Hash,
	minSafety types.SafetyLevel, executingDescriptor types.ExecutingDescriptor) error {

	err := f.backend.CheckAccessList(ctx, inboxEntries, minSafety, executingDescriptor)
	if err != nil {
		// Map errors to appropriate RPC error codes
		code := types.GetErrorCode(err)
		if code != 0 {
			return &rpc.JsonError{
				Code:    code,
				Message: err.Error(),
			}
		}
		// For unknown errors, return as-is (will be internal error)
		return err
	}
	return nil
}

// AdminFrontend handles admin RPC methods
type AdminFrontend struct {
	backend *Backend
}

// GetFailsafeEnabled returns whether failsafe is enabled
func (a *AdminFrontend) GetFailsafeEnabled(ctx context.Context) (bool, error) {
	return a.backend.FailsafeEnabled(), nil
}

// SetFailsafeEnabled is not supported - failsafe is automatically managed
func (a *AdminFrontend) SetFailsafeEnabled(ctx context.Context, enabled bool) error {
	return errors.New("SetFailsafeEnabled not supported: failsafe is automatically managed based on reorg detection")
}
