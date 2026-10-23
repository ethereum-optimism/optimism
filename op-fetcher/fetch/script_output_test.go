package fetch

import (
	"context"
	"encoding/json"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-fetcher/chaininfo"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/require"
)

// scriptOutput is an output of the FetchChainInfo forge script for a live chain, recorded at
// a pinned L1 block.
type scriptOutput struct {
	Network string         `json:"network"`
	Chain   string         `json:"chain"`
	Block   uint64         `json:"block"`
	Input   chainInfoInput `json:"input"`
	Output  chaininfo.Info `json:"output"`
}

// TestFetchChainInfoMatchesScriptOutput is a differential test against the FetchChainInfo
// forge script on live chains: super-root games with and without a permissionless game.
//
// It needs archive L1 RPCs in MAINNET_RPC_URL and SEPOLIA_RPC_URL, and skips a network whose
// RPC is unset.
func TestFetchChainInfoMatchesScriptOutput(t *testing.T) {
	files, err := filepath.Glob("testdata/script-output/*.json")
	require.NoError(t, err)
	require.NotEmpty(t, files)

	for _, file := range files {
		data, err := os.ReadFile(file)
		require.NoError(t, err)
		var want scriptOutput
		require.NoError(t, json.Unmarshal(data, &want))

		t.Run(want.Network+"/"+want.Chain, func(t *testing.T) {
			t.Parallel()
			rpcEnv := strings.ToUpper(want.Network) + "_RPC_URL"
			rpcURL := os.Getenv(rpcEnv)
			if rpcURL == "" {
				t.Skipf("%s not set", rpcEnv)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer cancel()
			client, err := ethclient.DialContext(ctx, rpcURL)
			require.NoError(t, err)
			defer client.Close()

			l1, _, err := newRPCL1State(ctx, client, new(big.Int).SetUint64(want.Block))
			require.NoError(t, err)
			got, err := fetchChainInfo(ctx, l1, want.Input.SystemConfigProxy, want.Input.L1StandardBridgeProxy)
			require.NoError(t, err)
			require.Equal(t, want.Output, got)
		})
	}
}
