package fetch

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/env"
	"github.com/ethereum-optimism/optimism/op-fetcher/pkg/fetcher/fetch/script"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

// TestFetchChainInfoEngineParity runs FetchChainInfo against OP Sepolia on both script engines and
// requires identical outputs: the Rust engine's fork mode is op-fetcher's default, the Go forked
// host is the --script-engine=go fallback.
func TestFetchChainInfoEngineParity(t *testing.T) {
	t.Parallel()

	l1RPCUrl := os.Getenv("SEPOLIA_RPC_URL")
	require.NotEmpty(t, l1RPCUrl, "SEPOLIA_RPC_URL must be set")

	// OP Sepolia (superchain-registry configs/sepolia/op.toml)
	systemConfigProxy := common.HexToAddress("0x034edD2A225f7f429A63E0f1D2084B9E0A93b538")
	l1StandardBridgeProxy := common.HexToAddress("0xFBb0621E0B23b5478B630BD55a5f21f67730B0F1")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	lgr := testlog.Logger(t, slog.LevelInfo)

	run := func(kind env.ScriptEngineKind) script.FetchChainInfoOutput {
		fetcher, err := NewFetcher(lgr, l1RPCUrl, systemConfigProxy, l1StandardBridgeProxy)
		require.NoError(t, err)
		fetcher.ScriptEngine = kind
		out, err := fetcher.FetchChainInfo(ctx)
		require.NoError(t, err, "FetchChainInfo failed on %q engine", kind)
		return out
	}

	goOut := run(env.ScriptEngineGo)
	rustOut := run(env.ScriptEngineRust)

	require.Equal(t, goOut, rustOut, "FetchChainInfo output must be identical across engines")

	// Non-vacuity: the output must carry real chain info, not zero values.
	require.Equal(t, systemConfigProxy, rustOut.SystemConfigProxy)
	require.NotEqual(t, common.Address{}, rustOut.OptimismPortalProxy, "portal proxy must be resolved")
	require.NotEqual(t, common.Address{}, rustOut.OpChainGuardian, "guardian role must be resolved")
}
