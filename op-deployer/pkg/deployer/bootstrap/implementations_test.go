package bootstrap

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/ethereum-optimism/optimism/op-service/testutils/devnet"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/standard"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/testutil"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

// defaultFallbackBlock is used when fixtures don't have metadata yet.
// One block past the v6.0.0-rc.2 OPCM deployment on Sepolia.
const defaultFallbackBlock = 10101510

func TestImplementations(t *testing.T) {
	t.Parallel()

	testCacheDir := testutils.IsolatedTestDirWithAutoCleanup(t)

	lgr := testlog.Logger(t, slog.LevelDebug)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	fixturePath := devnet.RPCReplayFixturePath("bootstrap_implementations")
	forkedL1, stopL1, err := devnet.NewForkedSepoliaFromFixture(lgr, fixturePath, defaultFallbackBlock)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, stopL1())
	})
	l1RPC := forkedL1.RPCUrl()

	// Sepolia chain ID = 11155111
	superchain, err := standard.SuperchainFor(11155111)
	require.NoError(t, err)

	loc, _ := testutil.LocalArtifacts(t)

	proxyAdminOwner, err := standard.L1ProxyAdminOwner(11155111)
	require.NoError(t, err)
	deploy := func() opcm.DeployImplementationsOutput {
		out, err := Implementations(ctx, ImplementationsConfig{
			L1RPCUrl:                        l1RPC,
			PrivateKey:                      testutil.AnvilDefaultPrivateKey,
			ArtifactsLocator:                loc,
			Logger:                          lgr,
			WithdrawalDelaySeconds:          standard.WithdrawalDelaySeconds,
			MinProposalSizeBytes:            standard.MinProposalSizeBytes,
			ChallengePeriodSeconds:          standard.ChallengePeriodSeconds,
			ProofMaturityDelaySeconds:       standard.ProofMaturityDelaySeconds,
			DisputeGameFinalityDelaySeconds: standard.DisputeGameFinalityDelaySeconds,
			MIPSVersion:                     int(standard.MIPSVersion),
			DevFeatureBitmap:                common.Hash{},
			SuperchainConfigProxy:           superchain.SuperchainConfigAddr,
			ProtocolVersionsProxy:           superchain.ProtocolVersionsAddr,
			SuperchainProxyAdmin:            proxyAdminOwner,
			L1ProxyAdminOwner:               proxyAdminOwner,
			Challenger:                      common.Address{'C'},
			CacheDir:                        testCacheDir,
			FaultGameMaxGameDepth:           standard.DisputeMaxGameDepth,
			FaultGameSplitDepth:             standard.DisputeSplitDepth,
			FaultGameClockExtension:         standard.DisputeClockExtension,
			FaultGameMaxClockDuration:       standard.DisputeMaxClockDuration,
		})
		require.NoError(t, err)
		return out
	}

	// Assert that addresses stay the same between runs
	t.Log("Deploying first implementation contracts bundle")
	deployment1 := deploy()
	require.NotEqual(t, common.Address{}, deployment1.Opcm, "Opcm address should be set")
	t.Log("Deploying second implementation contracts bundle")
	deployment2 := deploy()
	require.NotEqual(t, common.Address{}, deployment2.Opcm, "Opcm address should be set")
	require.Equal(t, deployment1, deployment2)
}
