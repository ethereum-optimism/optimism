package config

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"

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

func TestFaucetEntry_TxManagerConfig(t *testing.T) {
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

	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	addr := crypto.PubkeyToAddress(key.PublicKey)

	logger := testlog.Logger(t, log.LevelInfo)

	entry := FaucetEntry{
		ELRPC:   endpoint.MustRPC{Value: endpoint.URL("http://" + srv.Endpoint())},
		ChainID: eth.ChainIDFromUInt64(123),
		TxCfg: TxManagerConfig{
			PrivateKey: hexutil.Encode(crypto.FromECDSA(key)),
		},
	}
	t.Run("good chain", func(t *testing.T) {
		cfg, err := entry.TxManagerConfig(logger)
		require.NoError(t, err)
		require.Equal(t, chainID, eth.ChainIDFromBig(cfg.ChainID))
		require.Equal(t, addr, cfg.From)
	})

	t.Run("wrong chain", func(t *testing.T) {
		entry2 := entry
		entry2.ChainID = eth.ChainIDFromUInt64(4)
		_, err := entry2.TxManagerConfig(logger)
		require.ErrorContains(t, err, "unexpected chain ID")
	})

	t.Run("no key", func(t *testing.T) {
		entry2 := entry
		entry2.TxCfg.PrivateKey = ""
		_, err := entry2.TxManagerConfig(logger)
		require.ErrorContains(t, err, "could not init signer")
	})
}

func TestStaticConfigLoad(t *testing.T) {
	cfg := &Config{}
	result, err := cfg.Load(context.Background())
	require.NoError(t, err)
	require.Equal(t, cfg, result)
}
