package backend

import (
	"context"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/endpoint"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-sync-tester/metrics"
	stconf "github.com/ethereum-optimism/optimism/op-sync-tester/synctester/backend/config"

	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
)

type testAPI struct{}

func (b *testAPI) DummyAPI() {}

func TestBackend(t *testing.T) {
	srv := oprpc.NewServer("127.0.0.1", 0, "")
	srv.AddAPI(rpc.API{
		Namespace: "eth",
		Service:   &testAPI{},
	})
	require.NoError(t, srv.Start())
	t.Cleanup(func() {
		_ = srv.Stop()
	})

	logger := testlog.Logger(t, log.LevelInfo)

	cfg := &stconf.Config{
		ELRPC: endpoint.MustRPC{Value: endpoint.URL("http://" + srv.Endpoint())},
	}
	m := &metrics.NoopMetrics{}
	b, err := FromConfig(logger, m, cfg)
	require.NoError(t, err)

	require.Error(t, b.Init(context.Background()))

	session := &Session{Head: 3, Safe: 2, Finalized: 1}
	ctx := context.Background()
	ctx = context.WithValue(ctx, CtxKeySession, session)
	require.NoError(t, b.Init(ctx))

	_, ok := b.sessions[session.ID()]
	require.True(t, ok)

	b.ClearSessions(ctx)

	_, ok = b.sessions[session.ID()]
	require.False(t, ok)
}
