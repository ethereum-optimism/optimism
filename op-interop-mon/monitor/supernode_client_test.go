package monitor

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSupernodeClientCalls(t *testing.T) {
	rpc := &stubRPC{}
	sc := &SupernodeClient{client: rpc}
	_, _ = sc.SyncStatus(context.Background())
	_ = sc.Heartbeat(context.Background())
	require.Contains(t, rpc.calls, "supernode_syncStatus")
	require.Contains(t, rpc.calls, "heartbeat_check")
}
