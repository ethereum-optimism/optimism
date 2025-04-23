package backend

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"

	fconf "github.com/ethereum-optimism/optimism/op-faucet/faucet/backend/config"
	ftypes "github.com/ethereum-optimism/optimism/op-faucet/faucet/backend/types"
	"github.com/ethereum-optimism/optimism/op-faucet/metrics"
	"github.com/ethereum-optimism/optimism/op-service/endpoint"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

type testAPI struct {
	chainID eth.ChainID
}

func (b *testAPI) ChainId() *hexutil.Big {
	return (*hexutil.Big)(b.chainID.ToBig())
}

type testRouter struct {
	routes []string
	apis   map[string][]rpc.API
}

func (t *testRouter) AddRPC(route string) error {
	t.routes = append(t.routes, route)
	return nil
}

func (t *testRouter) AddAPIToRPC(route string, api rpc.API) error {
	t.apis[route] = append(t.apis[route], api)
	return nil
}

var _ APIRouter = (*testRouter)(nil)

func TestBackend(t *testing.T) {
	chainID := eth.ChainIDFromUInt64(123)
	srv := oprpc.NewServer("127.0.0.1", 0, "")
	srv.AddAPI(rpc.API{
		Namespace: "eth",
		Service:   &testAPI{chainID: chainID},
	})
	require.NoError(t, srv.Start())
	t.Cleanup(func() {
		_ = srv.Stop()
	})

	keyA, err := crypto.GenerateKey()
	require.NoError(t, err)
	keyB, err := crypto.GenerateKey()
	require.NoError(t, err)
	addrA := crypto.PubkeyToAddress(keyA.PublicKey)
	addrB := crypto.PubkeyToAddress(keyB.PublicKey)

	logger := testlog.Logger(t, log.LevelInfo)

	faucetCfgA := &fconf.FaucetEntry{
		ELRPC:   endpoint.MustRPC{Value: endpoint.URL("http://" + srv.Endpoint())},
		ChainID: eth.ChainIDFromUInt64(123),
		TxCfg: fconf.TxManagerConfig{
			PrivateKey: hexutil.Encode(crypto.FromECDSA(keyA)),
		},
	}
	faucetA := ftypes.FaucetID("faucetA")

	faucetCfgB := &fconf.FaucetEntry{
		ELRPC:   endpoint.MustRPC{Value: endpoint.URL("http://" + srv.Endpoint())},
		ChainID: eth.ChainIDFromUInt64(123),
		TxCfg: fconf.TxManagerConfig{
			PrivateKey: hexutil.Encode(crypto.FromECDSA(keyB)),
		},
	}
	faucetB := ftypes.FaucetID("faucetB")

	cfg := &fconf.Config{
		Faucets: map[ftypes.FaucetID]*fconf.FaucetEntry{
			faucetA: faucetCfgA,
			faucetB: faucetCfgB,
		},
		Defaults: map[eth.ChainID]ftypes.FaucetID{
			chainID: faucetA,
		},
	}
	m := &metrics.NoopMetrics{}
	r := &testRouter{
		routes: make([]string, 0),
		apis:   make(map[string][]rpc.API),
	}
	b, err := FromConfig(logger, m, cfg, r)
	require.NoError(t, err)

	require.Len(t, b.Faucets(), 2)
	require.Len(t, b.Defaults(), 1)

	fA := b.FaucetByChain(chainID)
	require.Equal(t, addrA, fA.txMgr.From())
	require.Equal(t, chainID, fA.ChainID())

	fB := b.FaucetByID(faucetB)
	require.Equal(t, addrB, fB.txMgr.From())
	require.Equal(t, chainID, fB.ChainID())

	b.DisableFaucet(faucetB)
	require.True(t, fB.disabled)
	b.EnableFaucet(faucetB)
	require.False(t, fB.disabled)

	b.DisableFaucet("other")
	b.EnableFaucet("other") // unknown faucets are noop

	require.NoError(t, b.Stop(context.Background()))
}
