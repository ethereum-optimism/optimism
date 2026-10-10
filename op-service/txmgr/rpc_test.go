package txmgr

import (
	"context"
	"fmt"
	"math/big"
	"testing"

	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
)

func TestTxmgrRPC(t *testing.T) {
	minBaseFeeInit := big.NewInt(2000)
	minPriorityFeeInit := big.NewInt(1000)
	minBlobFeeInit := big.NewInt(3000)
	feeThresholdInit := big.NewInt(4000)
	rebroadcastIntervalInit := int64(25)
	bumpFeeRetryTimeInit := int64(100)

	cfg := configWithNumConfs(1)
	cfg.MinBaseFee.Store(minBaseFeeInit)
	cfg.MinTipCap.Store(minPriorityFeeInit)
	cfg.MinBlobTxFee.Store(minBlobFeeInit)
	cfg.FeeLimitThreshold.Store(feeThresholdInit)
	cfg.RebroadcastInterval.Store(rebroadcastIntervalInit)
	cfg.ResubmissionTimeout.Store(bumpFeeRetryTimeInit)

	h := newTestHarnessWithConfig(t, cfg)
	h.mgr.blobTipOracle = &mockBlobTipOracle{suggestedTip: big.NewInt(1)}

	appVersion := "test"
	server := oprpc.NewServer(
		"127.0.0.1",
		0,
		appVersion,
	)
	server.AddAPI(h.mgr.API())
	require.NoError(t, server.Start())
	defer func() {
		_ = server.Stop()
	}()

	rpcClient, err := rpc.Dial(fmt.Sprintf("http://%s", server.Endpoint()))
	require.NoError(t, err)

	type tcase struct {
		rpcMethod string
		initValue *big.Int
	}

	cases := []tcase{
		{"MinBaseFee", minBaseFeeInit},
		{"MinPriorityFee", minPriorityFeeInit},
		{"MinBlobFee", minBlobFeeInit},
		{"FeeThreshold", feeThresholdInit},
		{"RebroadcastInterval", big.NewInt(rebroadcastIntervalInit)},
		{"BumpFeeRetryTime", big.NewInt(bumpFeeRetryTimeInit)},
	}

	for _, tc := range cases {
		t.Run("Get|Set"+tc.rpcMethod, func(t *testing.T) {
			var res *big.Int

			require.NoError(t, rpcClient.Call(&res, "txmgr_get"+tc.rpcMethod))
			require.Equal(t, tc.initValue, res)

			newVal := new(big.Int)
			newVal.Add(tc.initValue, big.NewInt(1))

			require.NoError(t, rpcClient.Call(&res, "txmgr_set"+tc.rpcMethod, newVal))
			require.NoError(t, rpcClient.Call(&res, "txmgr_get"+tc.rpcMethod))
			require.Equal(t, newVal, res)
		})
	}

	t.Run("Get|SetBlobTipCapDynamic", func(t *testing.T) {
		var res bool

		require.NoError(t, rpcClient.Call(&res, "txmgr_getBlobTipCapDynamic"))
		require.False(t, res)

		require.NoError(t, rpcClient.Call(nil, "txmgr_setBlobTipCapDynamic", true))
		require.NoError(t, rpcClient.Call(&res, "txmgr_getBlobTipCapDynamic"))
		require.True(t, res)

		require.NoError(t, rpcClient.Call(nil, "txmgr_setBlobTipCapDynamic", false))
		require.NoError(t, rpcClient.Call(&res, "txmgr_getBlobTipCapDynamic"))
		require.False(t, res)
	})
}

func TestTxmgrRPC_SetBlobTipCapDynamicWithoutOracle(t *testing.T) {
	h := newTestHarnessWithConfig(t, configWithNumConfs(1))
	require.Nil(t, h.mgr.blobTipOracle)

	server := oprpc.NewServer("127.0.0.1", 0, "test")
	server.AddAPI(h.mgr.API())
	require.NoError(t, server.Start())
	defer func() {
		_ = server.Stop()
	}()

	rpcClient, err := rpc.Dial(fmt.Sprintf("http://%s", server.Endpoint()))
	require.NoError(t, err)

	err = rpcClient.Call(nil, "txmgr_setBlobTipCapDynamic", true)
	require.ErrorContains(t, err, ErrNoBlobTipOracle.Error())

	var res bool
	require.NoError(t, rpcClient.Call(&res, "txmgr_getBlobTipCapDynamic"))
	require.False(t, res)
	require.NoError(t, rpcClient.Call(nil, "txmgr_setBlobTipCapDynamic", false))

	// Gas price suggestions keep working instead of dereferencing the missing oracle.
	_, _, _, _, err = h.mgr.SuggestGasPriceCaps(context.Background())
	require.NoError(t, err)
}
