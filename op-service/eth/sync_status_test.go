package eth

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestSyncStatusJSONHeads pins the L2 head fields of the optimism_syncStatus response.
// There is a single unsafe head, so no cross-unsafe head is reported.
func TestSyncStatusJSONHeads(t *testing.T) {
	data, err := json.Marshal(SyncStatus{})
	require.NoError(t, err)

	var fields map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(data, &fields))

	require.Contains(t, fields, "unsafe_l2")
	require.Contains(t, fields, "pending_safe_l2")
	require.Contains(t, fields, "local_safe_l2")
	require.Contains(t, fields, "safe_l2")
	require.Contains(t, fields, "finalized_l2")
	require.NotContains(t, fields, "cross_unsafe_l2")
}
