package txmgr

import (
	"fmt"
	"math/big"
	"testing"

	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
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
	server := oprpc.NewServer(
		"127.0.0.1",
		0,
		appVersion,
		oprpc.WithAPIs([]rpc.API{
			h.mgr.API(),
		}),
	)
	require.NoError(t, server.Start())
	defer func() {
		_ = server.Stop()
	}()

	rpcClient, err := rpc.Dial(fmt.Sprintf("http://%s", server.Endpoint()))
	require.NoError(t, err)

	type tcase struct {
		rpcMethod string
		value     *big.Int
	}

	cases := []tcase{
		{"MinBaseFee", big.NewInt(1001)},
		{"PriorityFee", big.NewInt(2001)},
		{"MinBlobFee", big.NewInt(3001)},
		{"FeeThreshold", big.NewInt(4001)},
	}

	for _, tc := range cases {
		t.Run(tc.rpcMethod, func(t *testing.T) {
			var res *big.Int
			require.NoError(t, rpcClient.Call(&res, "txmgr_set"+tc.rpcMethod, tc.value))
			require.NoError(t, rpcClient.Call(&res, "txmgr_get"+tc.rpcMethod))
			require.Equal(t, tc.value, res)
		})
	}
}
