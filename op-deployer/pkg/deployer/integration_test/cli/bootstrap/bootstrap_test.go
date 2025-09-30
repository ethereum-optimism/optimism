package bootstrap

import (
	"context"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/artifacts"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/bootstrap"
	test "github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/integration_test/cli"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/pipeline"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/standard"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/testutil"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

// TestEndToEndBootstrapApply tests that a system can be fully bootstrapped and applied, both from
// local artifacts and the default tagged artifacts. The tagged artifacts test only runs on proposal
// or backports branches, since those are the only branches with an SLA to support tagged artifacts.
func TestEndToEndBootstrapApply(t *testing.T) {
	require := require.New(t)

	lgr, l1RPC, l1Client, pkHex, pk, dk, l1ChainID := test.SetupAnvilTest(t)
	l2ChainID := uint256.NewInt(1)
	testCacheDir := testutils.IsolatedTestDirWithAutoCleanup(t)
	superchainPAO := common.Address{'S', 'P', 'A', 'O'}

	apply := func(t *testing.T, loc *artifacts.Locator) {
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()

		bstrap, err := bootstrap.Superchain(ctx, bootstrap.SuperchainConfig{
			L1RPCUrl:                   l1RPC,
			PrivateKey:                 pkHex,
			Logger:                     lgr,
			ArtifactsLocator:           loc,
			CacheDir:                   testCacheDir,
			SuperchainProxyAdminOwner:  superchainPAO,
			ProtocolVersionsOwner:      common.Address{'P', 'V', 'O'},
			Guardian:                   common.Address{'G'},
			Paused:                     false,
			RecommendedProtocolVersion: params.ProtocolVersion{0x01, 0x02, 0x03, 0x04},
			RequiredProtocolVersion:    params.ProtocolVersion{0x01, 0x02, 0x03, 0x04},
		})
		require.NoError(err)

		impls, err := bootstrap.Implementations(ctx, bootstrap.ImplementationsConfig{
			L1RPCUrl:                        l1RPC,
			PrivateKey:                      pkHex,
			ArtifactsLocator:                loc,
			MIPSVersion:                     int(standard.MIPSVersion),
			WithdrawalDelaySeconds:          standard.WithdrawalDelaySeconds,
			MinProposalSizeBytes:            standard.MinProposalSizeBytes,
			ChallengePeriodSeconds:          standard.ChallengePeriodSeconds,
			ProofMaturityDelaySeconds:       standard.ProofMaturityDelaySeconds,
			DisputeGameFinalityDelaySeconds: standard.DisputeGameFinalityDelaySeconds,
			DevFeatureBitmap:                common.Hash{},
			SuperchainConfigProxy:           bstrap.SuperchainConfigProxy,
			ProtocolVersionsProxy:           bstrap.ProtocolVersionsProxy,
			UpgradeController:               superchainPAO,
			SuperchainProxyAdmin:            bstrap.SuperchainProxyAdmin,
			CacheDir:                        testCacheDir,
			Logger:                          lgr,
			Challenger:                      common.Address{'C'},
		})
		require.NoError(err)

		intent, st := test.NewIntent(t, l1ChainID, dk, l2ChainID, loc, loc)
		intent.SuperchainRoles = nil
		intent.OPCMAddress = &impls.Opcm

		require.NoError(deployer.ApplyPipeline(
			ctx,
			deployer.ApplyPipelineOpts{
				DeploymentTarget:   deployer.DeploymentTargetLive,
				L1RPCUrl:           l1RPC,
				DeployerPrivateKey: pk,
				Intent:             intent,
				State:              st,
				Logger:             lgr,
				StateWriter:        pipeline.NoopStateWriter(),
				CacheDir:           testCacheDir,
			},
		))

		cg := test.EthClientCodeGetter(ctx, l1Client)
		validateSuperchainDeployment(t, st, cg, true)
		validateOPChainDeployment(t, cg, st)
	}

	t.Run("default tagged artifacts", func(t *testing.T) {
		apply(t, artifacts.DefaultL1ContractsLocator)
	})

	t.Run("local artifacts", func(t *testing.T) {
		loc, _ := testutil.LocalArtifacts(t)
		apply(t, loc)
	})
}

// Validate that the superchain addresses are always set, even in subsequent deployments
// that pull from an existing OPCM deployment.
func validateSuperchainDeployment(t *testing.T, st *state.State, cg test.CodeGetter, includeSuperchainImpls bool) {
	type addrTuple struct {
		name string
		addr common.Address
	}
	addrs := []addrTuple{
		{"SuperchainProxyAdminImpl", st.SuperchainDeployment.SuperchainProxyAdminImpl},
		{"SuperchainConfigProxy", st.SuperchainDeployment.SuperchainConfigProxy},
		{"ProtocolVersionsProxy", st.SuperchainDeployment.ProtocolVersionsProxy},
		{"OpcmImpl", st.ImplementationsDeployment.OpcmImpl},
		{"PreimageOracleImpl", st.ImplementationsDeployment.PreimageOracleImpl},
		{"MipsImpl", st.ImplementationsDeployment.MipsImpl},
	}

	if includeSuperchainImpls {
		addrs = append(addrs, addrTuple{"SuperchainConfigImpl", st.SuperchainDeployment.SuperchainConfigImpl})
		addrs = append(addrs, addrTuple{"ProtocolVersionsImpl", st.SuperchainDeployment.ProtocolVersionsImpl})
	}

	for _, addr := range addrs {
		t.Run(addr.name, func(t *testing.T) {
			code := cg(t, addr.addr)
			require.NotEmpty(t, code, "contract %s at %s has no code", addr.name, addr.addr)
		})
	}
}

// Validate that the implementation addresses are always set, even in subsequent deployments
// that pull from an existing OPCM deployment.
func validateOPChainDeployment(t *testing.T, cg test.CodeGetter, st *state.State) {
	type addrTuple struct {
		name string
		addr common.Address
	}
	implAddrs := []addrTuple{
		{"DelayedWethImpl", st.ImplementationsDeployment.DelayedWethImpl},
		{"OptimismPortalImpl", st.ImplementationsDeployment.OptimismPortalImpl},
		{"OptimismPortalInteropImpl", st.ImplementationsDeployment.OptimismPortalInteropImpl},
		{"SystemConfigImpl", st.ImplementationsDeployment.SystemConfigImpl},
		{"L1CrossDomainMessengerImpl", st.ImplementationsDeployment.L1CrossDomainMessengerImpl},
		{"L1ERC721BridgeImpl", st.ImplementationsDeployment.L1Erc721BridgeImpl},
		{"L1StandardBridgeImpl", st.ImplementationsDeployment.L1StandardBridgeImpl},
		{"OptimismMintableERC20FactoryImpl", st.ImplementationsDeployment.OptimismMintableErc20FactoryImpl},
		{"DisputeGameFactoryImpl", st.ImplementationsDeployment.DisputeGameFactoryImpl},
		{"MipsImpl", st.ImplementationsDeployment.MipsImpl},
		{"PreimageOracleImpl", st.ImplementationsDeployment.PreimageOracleImpl},
	}

	for _, addr := range implAddrs {
		require.NotEmpty(t, addr.addr, "%s should be set", addr.name)
		code := cg(t, addr.addr)
		require.NotEmpty(t, code, "contract %s at %s has no code", addr.name, addr.addr)
	}
}
