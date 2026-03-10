package filter

import (
	"context"
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-service/eth"
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
		return &rpc.JsonError{
			Code:    types.GetErrorCode(err),
			Message: err.Error(),
		}
	}
	return nil
}

// PublicAdminFrontend exposes read-only admin methods on the public port.
type PublicAdminFrontend struct {
	backend *Backend
}

func (p *PublicAdminFrontend) GetFailsafeEnabled(ctx context.Context) (bool, error) {
	return p.backend.FailsafeEnabled(), nil
}

// AdminFrontend handles admin RPC methods
type AdminFrontend struct {
	backend *Backend
}

// GetFailsafeEnabled returns whether failsafe is enabled
func (a *AdminFrontend) GetFailsafeEnabled(ctx context.Context) (bool, error) {
	return a.backend.FailsafeEnabled(), nil
}

// SetFailsafeEnabled enables or disables failsafe mode
func (a *AdminFrontend) SetFailsafeEnabled(ctx context.Context, enabled bool) error {
	a.backend.SetFailsafeEnabled(enabled)
	return nil
}

// Rewind is not implemented. For cross-chain consistency, rewind would need to
// coordinate across all chains to the same timestamp, which is complex.
// Instead, wipe the filter's data directory and restart fresh.
func (a *AdminFrontend) Rewind(ctx context.Context, chain eth.ChainID, block eth.BlockID) error {
	return errors.New("rewind not implemented: wipe the filter's data directory and restart instead")
}
