package txmgr

import (
	"fmt"
	"io"
	"math/big"
	"net/http"
	"testing"

	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
)

func TestTxmgrRPC(t *testing.T) {
	minBaseFee := big.NewInt(1000)
	priorityFee := big.NewInt(2000)
	minBlobFee := big.NewInt(3000)
	feeThreshold := big.NewInt(4000)

	cfg := Config{
		MinBaseFee:        minBaseFee,
		MinTipCap:         priorityFee,
		MinBlobTxFee:      minBlobFee,
		FeeLimitThreshold: feeThreshold,
	}
	h := newTestHarnessWithConfig(t, cfg)

	appVersion := "test"
	m := &testutils.TestRPCMetrics{}
	l := testlog.Logger(t, log.LevelDebug)

	server := oprpc.NewServer(
		"127.0.0.1",
		0,
		appVersion,
		oprpc.WithAPIs([]rpc.API{
			NewTxmgrApi(h.mgr, m, l),
		}),
	)
	require.NoError(t, server.Start())
	defer func() {
		_ = server.Stop()
	}()

	rpcClient, err := rpc.Dial(fmt.Sprintf("http://%s", server.Endpoint()))
	require.NoError(t, err)

	t.Run("supports GET /healthz", func(t *testing.T) {
		res, err := http.Get(fmt.Sprintf("http://%s/healthz", server.Endpoint()))
		require.NoError(t, err)
		defer res.Body.Close()
		body, err := io.ReadAll(res.Body)
		require.NoError(t, err)
		require.EqualValues(t, fmt.Sprintf("{\"version\":\"%s\"}\n", appVersion), string(body))
	})

	t.Run("supports health_status", func(t *testing.T) {
		var res string
		require.NoError(t, rpcClient.Call(&res, "health_status"))
		require.Equal(t, appVersion, res)
	})

	t.Run("txmgr_getMinBaseFee", func(t *testing.T) {
		var res *big.Int
		require.NoError(t, rpcClient.Call(&res, "txmgr_getMinBaseFee"))
		require.Equal(t, minBaseFee, res)
	})

	t.Run("txmgr_setMinBaseFee", func(t *testing.T) {
		var res *big.Int
		minBaseFee.Add(minBaseFee, big.NewInt(1))
		require.NoError(t, rpcClient.Call(&res, "txmgr_setMinBaseFee", minBaseFee))
		require.NoError(t, rpcClient.Call(&res, "txmgr_getMinBaseFee"))
		require.Equal(t, minBaseFee, res)
	})

	t.Run("txmgr_getPriorityFee", func(t *testing.T) {
		var res *big.Int
		require.NoError(t, rpcClient.Call(&res, "txmgr_getPriorityFee"))
		require.Equal(t, priorityFee, res)
	})

	t.Run("txmgr_setPriorityFee", func(t *testing.T) {
		var res *big.Int
		priorityFee.Add(priorityFee, big.NewInt(1))
		require.NoError(t, rpcClient.Call(&res, "txmgr_setPriorityFee", priorityFee))
		require.NoError(t, rpcClient.Call(&res, "txmgr_getPriorityFee"))
		require.Equal(t, priorityFee, res)
	})

	t.Run("txmgr_getMinBlobFee", func(t *testing.T) {
		var res *big.Int
		require.NoError(t, rpcClient.Call(&res, "txmgr_getMinBlobFee"))
		require.Equal(t, minBlobFee, res)
	})

	t.Run("txmgr_setMinBlobFee", func(t *testing.T) {
		var res *big.Int
		minBlobFee.Add(minBlobFee, big.NewInt(1))
		require.NoError(t, rpcClient.Call(&res, "txmgr_setMinBlobFee", minBlobFee))
		require.NoError(t, rpcClient.Call(&res, "txmgr_getMinBlobFee"))
		require.Equal(t, minBlobFee, res)
	})

	t.Run("txmgr_getFeeThreshold", func(t *testing.T) {
		var res *big.Int
		require.NoError(t, rpcClient.Call(&res, "txmgr_getFeeThreshold"))
		require.Equal(t, feeThreshold, res)
	})

	t.Run("txmgr_setFeeThreshold", func(t *testing.T) {
		var res *big.Int
		feeThreshold.Add(feeThreshold, big.NewInt(1))
		require.NoError(t, rpcClient.Call(&res, "txmgr_setFeeThreshold", feeThreshold))
		require.NoError(t, rpcClient.Call(&res, "txmgr_getFeeThreshold"))
		require.Equal(t, feeThreshold, res)
	})
}
