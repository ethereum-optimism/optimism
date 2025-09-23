package forge

import (
	"context"
	"fmt"
	"log/slog"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/artifacts"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/bootstrap"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/forge"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/standard"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/ethereum-optimism/optimism/op-service/testutils/devnet"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
)

func TestPipelineForgeApply(t *testing.T) {
	lgr := testlog.Logger(t, slog.LevelInfo)
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()

	// Start Anvil
	l1RPC, l1Client := devnet.DefaultAnvilRPC(t, lgr)
	defer l1Client.Close()

	pkHex := "ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
	pk, err := crypto.HexToECDSA(pkHex)
	require.NoError(t, err)
	sender := crypto.PubkeyToAddress(pk.PublicKey)

	testCacheDir := testutils.IsolatedTestDirWithAutoCleanup(t)
	embeddedArtifactsFS, err := artifacts.ExtractEmbedded(testCacheDir)
	require.NoError(t, err)

	// Bootstrap superchain to the RPC
	bstrap, err := bootstrap.Superchain(ctx, bootstrap.SuperchainConfig{
		L1RPCUrl:                   l1RPC,
		PrivateKey:                 pkHex,
		Logger:                     lgr,
		ArtifactsLocator:           artifacts.EmbeddedLocator,
		CacheDir:                   testCacheDir,
		SuperchainProxyAdminOwner:  common.Address{'S', 'P', 'A', 'O'},
		ProtocolVersionsOwner:      common.Address{'P', 'V', 'O'},
		Guardian:                   common.Address{'G'},
		Paused:                     false,
		RecommendedProtocolVersion: params.ProtocolVersion{0x01, 0x02, 0x03, 0x04},
		RequiredProtocolVersion:    params.ProtocolVersion{0x01, 0x02, 0x03, 0x04},
	})
	require.NoError(t, err)
	require.NotEqual(t, common.Address{}, bstrap.SuperchainConfigProxy)
	require.NotEqual(t, common.Address{}, bstrap.ProtocolVersionsProxy)

	forgeClient, err := forge.NewStandardClient(fmt.Sprintf("%v", embeddedArtifactsFS))
	require.NoError(t, err)

	// Deploy Implementations via Forge
	implInput := opcm.DeployImplementationsInput{
		WithdrawalDelaySeconds:          new(big.Int).SetUint64(standard.WithdrawalDelaySeconds),
		MinProposalSizeBytes:            new(big.Int).SetUint64(standard.MinProposalSizeBytes),
		ChallengePeriodSeconds:          new(big.Int).SetUint64(standard.ChallengePeriodSeconds),
		ProofMaturityDelaySeconds:       new(big.Int).SetUint64(standard.ProofMaturityDelaySeconds),
		DisputeGameFinalityDelaySeconds: new(big.Int).SetUint64(standard.DisputeGameFinalityDelaySeconds),
		MipsVersion:                     new(big.Int).SetUint64(standard.MIPSVersion),
		DevFeatureBitmap:                common.Hash{},
		SuperchainConfigProxy:           bstrap.SuperchainConfigProxy,
		ProtocolVersionsProxy:           bstrap.ProtocolVersionsProxy,
		SuperchainProxyAdmin:            bstrap.SuperchainProxyAdmin,
		UpgradeController:               common.Address{'S', 'P', 'A', 'O'},
		Challenger:                      common.Address{'C'},
	}
	implOut, _, err := opcm.NewDeployImplementationsForgeCaller(forgeClient)(ctx, implInput)
	require.NoError(t, err)
	require.NotEqual(t, common.Address{}, implOut.Opcm)

	// Deploy OPChain
	opInput := opcm.DeployOPChainInput{
		OpChainProxyAdminOwner:       sender,
		SystemConfigOwner:            sender,
		Batcher:                      sender,
		UnsafeBlockSigner:            sender,
		Proposer:                     sender,
		Challenger:                   sender,
		BasefeeScalar:                1000,
		BlobBaseFeeScalar:            1360,
		L2ChainId:                    big.NewInt(1_000_000_001),
		Opcm:                         implOut.Opcm,
		SaltMixer:                    "mix", // non-empty salt mixer
		GasLimit:                     standard.GasLimit,
		DisputeGameType:              standard.DisputeGameType,
		DisputeAbsolutePrestate:      standard.DisputeAbsolutePrestate,
		DisputeMaxGameDepth:          new(big.Int).SetUint64(standard.DisputeMaxGameDepth),
		DisputeSplitDepth:            new(big.Int).SetUint64(standard.DisputeSplitDepth),
		DisputeClockExtension:        standard.DisputeClockExtension,
		DisputeMaxClockDuration:      standard.DisputeMaxClockDuration,
		AllowCustomDisputeParameters: false,
		OperatorFeeScalar:            0,
		OperatorFeeConstant:          0,
	}
	opOut, _, err := opcm.NewDeployOPChainForgeCaller(forgeClient)(ctx, opInput)
	require.NoError(t, err)
	require.NotEqual(t, common.Address{}, opOut.OpChainProxyAdmin)
	require.NotEqual(t, common.Address{}, opOut.SystemConfigProxy)
	require.NotEqual(t, common.Address{}, opOut.OptimismPortalProxy)
}
