package interop

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestFailsafeErrorCode verifies failsafe has a dedicated RPC error code that
// clients can detect by code rather than by matching the message string. This
// code is a deployment contract with op-reth and proxyd, so it must not change.
func TestFailsafeErrorCode(t *testing.T) {
	require.Equal(t, -320602, GetErrorCode(ErrFailsafeEnabled),
		"failsafe must map to the dedicated -320602 code")

	require.NotEqual(t, genericInvalidParamsErr, GetErrorCode(ErrFailsafeEnabled),
		"failsafe must not share the generic invalid-params fallback code")
}

// TestFailsafeErrorCodeWrapped confirms detection survives error wrapping, since
// GetErrorCode uses errors.Is and callers wrap the sentinel with context.
func TestFailsafeErrorCodeWrapped(t *testing.T) {
	wrapped := fmt.Errorf("rejecting request: %w", ErrFailsafeEnabled)
	require.Equal(t, -320602, GetErrorCode(wrapped))
}
