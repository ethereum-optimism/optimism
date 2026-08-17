package bootstrap

import (
	"context"
	"log/slog"
	"math/big"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-core/devfeatures"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/artifacts"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/ethereum-optimism/optimism/op-service/testutils/devnet"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/standard"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/testutil"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/require"
)

var networks = []string{"mainnet", "sepolia"}

func TestImplementationsConfigSP1Verifier(t *testing.T) {
	valid := ImplementationsConfig{
		L1RPCUrl:                        "http://localhost:8545",
		PrivateKey:                      testutil.AnvilDefaultPrivateKey,
		ArtifactsLocator:                artifacts.EmbeddedLocator,
		MIPSVersion:                     int(standard.MIPSVersion),
		WithdrawalDelaySeconds:          1,
		MinProposalSizeBytes:            1,
		ChallengePeriodSeconds:          1,
		ProofMaturityDelaySeconds:       1,
		DisputeGameFinalityDelaySeconds: 1,
		FaultGameMaxGameDepth:           1,
		FaultGameSplitDepth:             1,
		FaultGameClockExtension:         1,
		FaultGameMaxClockDuration:       1,
		SuperchainConfigProxy:           common.Address{0x01},
		L1ProxyAdminOwner:               common.Address{0x02},
		SuperchainProxyAdmin:            common.Address{0x03},
		Challenger:                      common.Address{0x04},
		Logger:                          testlog.Logger(t, slog.LevelDebug),
	}

	t.Run("disabled requires zero verifier", func(t *testing.T) {
		cfg := valid
		cfg.SP1Verifier = common.Address{0x05}
		require.ErrorContains(t, cfg.Check(), "must not be specified")
	})

	t.Run("enabled accepts network default", func(t *testing.T) {
		cfg := valid
		cfg.DevFeatureBitmap = devfeatures.ZKDisputeGameFlag
		require.NoError(t, cfg.Check())
	})

	t.Run("enabled accepts verifier", func(t *testing.T) {
		cfg := valid
		cfg.DevFeatureBitmap = devfeatures.ZKDisputeGameFlag
		cfg.SP1Verifier = common.Address{0x05}
		require.NoError(t, cfg.Check())
	})
}

func TestImplementationsConfigResolveSP1Verifier(t *testing.T) {
	t.Run("mainnet default", func(t *testing.T) {
		cfg := ImplementationsConfig{DevFeatureBitmap: devfeatures.ZKDisputeGameFlag}
		require.NoError(t, cfg.resolveSP1Verifier(big.NewInt(1)))
		require.Equal(t, common.HexToAddress("0xc3c6dDDAc8829b233Dc6536Ec024775a57b0AF2A"), cfg.SP1Verifier)
	})

	t.Run("sepolia default", func(t *testing.T) {
		cfg := ImplementationsConfig{DevFeatureBitmap: devfeatures.ZKDisputeGameFlag}
		require.NoError(t, cfg.resolveSP1Verifier(big.NewInt(11155111)))
		require.Equal(t, common.HexToAddress("0xc3c6dDDAc8829b233Dc6536Ec024775a57b0AF2A"), cfg.SP1Verifier)
	})

	t.Run("unknown network fails", func(t *testing.T) {
		cfg := ImplementationsConfig{DevFeatureBitmap: devfeatures.ZKDisputeGameFlag}
		err := cfg.resolveSP1Verifier(big.NewInt(900))
		require.ErrorContains(t, err, "no default SP1 verifier for L1 chain ID 900")
		require.ErrorContains(t, err, "--sp1-verifier-address")
	})

	t.Run("large unknown network fails", func(t *testing.T) {
		cfg := ImplementationsConfig{DevFeatureBitmap: devfeatures.ZKDisputeGameFlag}
		chainID := new(big.Int).Lsh(big.NewInt(1), 80)
		err := cfg.resolveSP1Verifier(chainID)
		require.ErrorContains(t, err, "no default SP1 verifier for L1 chain ID")
		require.ErrorContains(t, err, "--sp1-verifier-address")
	})

	t.Run("override permits unknown network", func(t *testing.T) {
		override := common.Address{0x05}
		cfg := ImplementationsConfig{
			DevFeatureBitmap: devfeatures.ZKDisputeGameFlag,
			SP1Verifier:      override,
		}
		require.NoError(t, cfg.resolveSP1Verifier(big.NewInt(900)))
		require.Equal(t, override, cfg.SP1Verifier)
	})

	t.Run("disabled does not select verifier", func(t *testing.T) {
		cfg := ImplementationsConfig{}
		require.NoError(t, cfg.resolveSP1Verifier(big.NewInt(900)))
		require.Equal(t, common.Address{}, cfg.SP1Verifier)
	})
}

func TestImplementations(t *testing.T) {
	for _, network := range networks {
		t.Run(network, func(t *testing.T) {
			envVar := strings.ToUpper(network) + "_RPC_URL"
			rpcURL := os.Getenv(envVar)
			require.NotEmpty(t, rpcURL, "must specify RPC url via %s env var", envVar)
			testImplementations(t, rpcURL)
		})
	}
}

func testImplementations(t *testing.T, forkRPCURL string) {
	testCacheDir := testutils.IsolatedTestDirWithAutoCleanup(t)

	if forkRPCURL == "" {
		t.Skip("forkRPCURL not set")
	}

	lgr := testlog.Logger(t, slog.LevelDebug)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	forkedL1, stopL1, err := devnet.NewForked(lgr, forkRPCURL)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, stopL1())
	})
	l1RPC := forkedL1.RPCUrl()

	client, err := ethclient.Dial(l1RPC)
	require.NoError(t, err)

	chainID, err := client.ChainID(ctx)
	require.NoError(t, err)

	superchain, err := standard.SuperchainFor(bigs.Uint64Strict(chainID))
	require.NoError(t, err)

	loc, _ := testutil.LocalArtifacts(t)

	proxyAdminOwner, err := standard.L1ProxyAdminOwner(bigs.Uint64Strict(chainID))
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
	require.NotEqual(t, common.Address{}, deployment1.OpcmV2, "OpcmV2 address should be set")
	t.Log("Deploying second implementation contracts bundle")
	deployment2 := deploy()
	require.NotEqual(t, common.Address{}, deployment2.OpcmV2, "OpcmV2 address should be set")
	require.Equal(t, deployment1, deployment2)
}
