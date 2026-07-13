package backend

import (
	"fmt"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/endpoint"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-sync-tester/metrics"
	stconf "github.com/ethereum-optimism/optimism/op-sync-tester/synctester/backend/config"

	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
	sttypes "github.com/ethereum-optimism/optimism/op-sync-tester/synctester/backend/types"
)

type testAPI struct{}

func (b *testAPI) DummyAPI() {}

type testRouter struct {
	routes    []string
	apis      map[string][]rpc.API
	addRPCErr func(route string) error
}

func (t *testRouter) AddRPC(route string) error {
	t.routes = append(t.routes, route)
	if t.addRPCErr != nil {
		return t.addRPCErr(route)
	}
	return nil
}

func (t *testRouter) AddAPIToRPC(route string, api rpc.API) error {
	t.apis[route] = append(t.apis[route], api)
	return nil
}

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

	syncTesterCfgA := &stconf.SyncTesterEntry{
		ELRPC:   endpoint.MustRPC{Value: endpoint.URL("http://" + srv.Endpoint())},
		ChainID: eth.ChainIDFromUInt64(1),
	}
	syncTesterA := sttypes.SyncTesterID("syncTesterA")

	syncTesterCfgB := &stconf.SyncTesterEntry{
		ELRPC:   endpoint.MustRPC{Value: endpoint.URL("http://" + srv.Endpoint())},
		ChainID: eth.ChainIDFromUInt64(2),
	}
	syncTesterB := sttypes.SyncTesterID("syncTesterB")

	cfg := &stconf.Config{
		SyncTesters: map[sttypes.SyncTesterID]*stconf.SyncTesterEntry{
			syncTesterA: syncTesterCfgA,
			syncTesterB: syncTesterCfgB,
		},
	}
	m := &metrics.NoopMetrics{}
	r := &testRouter{
		routes: make([]string, 0),
		apis:   make(map[string][]rpc.API),
	}
	b, err := FromConfig(logger, m, cfg, r)
	require.NoError(t, err)

	require.Len(t, b.SyncTesters(), 2)
}

func TestBackendFromConfigJoinsRouteSetupErrors(t *testing.T) {
	srv := oprpc.NewServer("127.0.0.1", 0, "")
	srv.AddAPI(rpc.API{
		Namespace: "eth",
		Service:   &testAPI{},
	})
	require.NoError(t, srv.Start())
	t.Cleanup(func() {
		_ = srv.Stop()
	})

	cfg := &stconf.Config{
		SyncTesters: map[sttypes.SyncTesterID]*stconf.SyncTesterEntry{
			"syncTesterA": {
				ELRPC:   endpoint.MustRPC{Value: endpoint.URL("http://" + srv.Endpoint())},
				ChainID: eth.ChainIDFromUInt64(1),
			},
			"syncTesterB": {
				ELRPC:   endpoint.MustRPC{Value: endpoint.URL("http://" + srv.Endpoint())},
				ChainID: eth.ChainIDFromUInt64(2),
			},
		},
	}
	r := &testRouter{
		routes: make([]string, 0),
		apis:   make(map[string][]rpc.API),
		addRPCErr: func(route string) error {
			return fmt.Errorf("failed route %s", route)
		},
	}

	_, err := FromConfig(testlog.Logger(t, log.LevelInfo), &metrics.NoopMetrics{}, cfg, r)
	require.Error(t, err)
	require.ErrorContains(t, err, "/chain/1/synctest")
	require.ErrorContains(t, err, "/chain/2/synctest")
}
