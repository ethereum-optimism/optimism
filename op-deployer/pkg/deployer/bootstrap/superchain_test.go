package bootstrap

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/ethereum-optimism/optimism/op-service/testutils/devnet"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/artifacts"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/testutil"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
)

func TestSuperchain(t *testing.T) {
	t.Parallel()

	lgr := testlog.Logger(t, slog.LevelDebug)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	fixturePath := devnet.RPCReplayFixturePath("bootstrap_superchain")
	forkedL1, stopL1, err := devnet.NewForkedSepoliaFromFixture(lgr, fixturePath, defaultFallbackBlock)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, stopL1())
	})
	l1RPC := forkedL1.RPCUrl()

	testCacheDir := testutils.IsolatedTestDirWithAutoCleanup(t)

	out, err := Superchain(ctx, SuperchainConfig{
		L1RPCUrl:         l1RPC,
		PrivateKey:       testutil.AnvilDefaultPrivateKey,
		ArtifactsLocator: artifacts.EmbeddedLocator,
		Logger:           lgr,

		SuperchainProxyAdminOwner:  common.Address{'S'},
		ProtocolVersionsOwner:      common.Address{'P'},
		Guardian:                   common.Address{'G'},
		Paused:                     false,
		RequiredProtocolVersion:    params.ProtocolVersionV0{Major: 1}.Encode(),
		RecommendedProtocolVersion: params.ProtocolVersionV0{Major: 2}.Encode(),
		CacheDir:                   testCacheDir,
	})
	require.NoError(t, err)

	client, err := ethclient.Dial(l1RPC)
	require.NoError(t, err)

	addresses := []common.Address{
		out.SuperchainConfigProxy,
		out.SuperchainConfigImpl,
		out.SuperchainProxyAdmin,
		out.ProtocolVersionsImpl,
		out.ProtocolVersionsProxy,
	}
	for _, addr := range addresses {
		require.NotEmpty(t, addr)

		code, err := client.CodeAt(ctx, addr, nil)
		require.NoError(t, err)
		require.NotEmpty(t, code)
	}
}
