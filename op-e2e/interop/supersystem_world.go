package interop

import (
	"context"
	"math/big"
	"path/filepath"
	"slices"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/params/forks"
	"github.com/holiman/uint256"
	"github.com/lmittmann/w3"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm/versions"
	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-core/devfeatures"
	opforks "github.com/ethereum-optimism/optimism/op-core/forks"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/artifacts"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/pipeline"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/intentbuilder"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/transactions"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// WorldConfig configures the chains of a SuperSystem.
type WorldConfig struct {
	L1ChainID  uint64
	L2ChainIDs []uint64
	// ContractArtifacts is the forge-artifacts directory of contracts-bedrock.
	ContractArtifacts string
}

// devUserCount is the number of prefunded user accounts on each chain.
const devUserCount = 20

var (
	// devAccountFunds is the balance of each prefunded dev account.
	devAccountFunds = new(uint256.Int).Mul(uint256.NewInt(10_000_000), uint256.NewInt(params.Ether))

	// konaAbsolutePrestate is the absolute prestate of the SUPER_CANNON_KONA game.
	konaAbsolutePrestate = common.HexToHash("0x035ef680a6fa34c50d8d8169075b5d133ecd7b38fe2b2a83cc76fc81ae5d7c52")

	// superCannonKonaInitBond matches DeployOPChain.DEFAULT_INIT_BOND.
	superCannonKonaInitBond = big.NewInt(8e16)

	migrateFn = w3.MustNewFunc(
		"migrate((address[] chainSystemConfigs,(bool enabled,uint256 initBond,uint32 gameType,bytes gameArgs)[] disputeGameConfigs,(bytes32 root,uint256 l2SequenceNumber) startingAnchorRoot,uint32 startingRespectedGameType))",
		"",
	)
	disputeGameFactoryFn = w3.MustNewFunc("disputeGameFactory()", "address")
)

// migrateInput is the input of OPContractsManagerV2.migrate.
type migrateInput struct {
	ChainSystemConfigs        []common.Address
	DisputeGameConfigs        []disputeGameConfig
	StartingAnchorRoot        opcm.Proposal
	StartingRespectedGameType uint32
}

type disputeGameConfig struct {
	Enabled  bool
	InitBond *big.Int
	GameType uint32
	GameArgs []byte
}

// prepareWorld deploys the superchain and the L2 chains into the L1 genesis with op-deployer.
func (s *interopE2ESystem) prepareWorld(w WorldConfig) {
	deployerKey, err := s.hdWallet.Secret(devkeys.SuperchainDeployerKey.Key(new(big.Int).SetUint64(w.L1ChainID)))
	require.NoError(s.t, err)
	st := &state.State{Version: 1}
	err = deployer.ApplyPipeline(context.Background(), deployer.ApplyPipelineOpts{
		DeploymentTarget:   deployer.DeploymentTargetGenesis,
		DeployerPrivateKey: deployerKey,
		Intent:             s.worldIntent(w),
		State:              st,
		Logger:             s.logger.New("role", "deployer"),
		StateWriter:        pipeline.NoopStateWriter(),
	})
	require.NoError(s.t, err, "failed to deploy the world")

	l1Genesis := *st.L1DevGenesis
	l1Genesis.StateHash = nil
	l1Genesis.Alloc = st.L1StateDump.Data.Accounts
	s.deployment = st
	s.l1Genesis = &l1Genesis
}

// worldIntent returns the intent for the chains of w. The intent does not set the L1 genesis
// timestamp, so op-deployer uses the time at which the deployment completes.
func (s *interopE2ESystem) worldIntent(w WorldConfig) *state.Intent {
	artifactsDir, err := filepath.Abs(w.ContractArtifacts)
	require.NoError(s.t, err)
	loc, err := artifacts.NewFileLocator(artifactsDir)
	require.NoError(s.t, err)

	builder := intentbuilder.New().
		WithL1ContractsLocator(loc).
		WithL2ContractsLocator(loc).
		WithUseInterop(true)
	for key, value := range map[string]any{
		"devFeatureBitmap":                devfeatures.EnableDevFeature(common.Hash{}, devfeatures.OptimismPortalInteropFlag),
		"mipsVersion":                     uint64(versions.GetExperimentalVersion()),
		"preimageOracleMinProposalSize":   10000,
		"preimageOracleChallengePeriod":   120,
		"proofMaturityDelaySeconds":       12,
		"disputeGameFinalityDelaySeconds": 6,
		// The implementations bound the delays above, so lower the bounds to admit them.
		"minProofMaturityDelaySeconds":       1,
		"minDisputeGameFinalityDelaySeconds": 1,
		"finalizationPeriodSeconds":          2,
		"maxSequencerDrift":                  300,
		"sequencerWindowSize":                200,
		"channelTimeout":                     120,
		"baseFeeVaultWithdrawalNetwork":      "remote",
		"l1FeeVaultWithdrawalNetwork":        "remote",
		"sequencerFeeVaultWithdrawalNetwork": "remote",
		"operatorFeeVaultWithdrawalNetwork":  "remote",
	} {
		builder.WithGlobalOverride(key, value)
	}

	l1ID := eth.ChainIDFromUInt64(w.L1ChainID)
	_, l1Cfg := builder.WithL1(l1ID)
	l1Cfg.WithGasLimit(30_000_000)
	// The interop migration runs on the L1 through an EIP-7702 delegation.
	l1Cfg.WithL1ForkAtGenesis(forks.Prague)

	keyAddr := intentbuilder.KeyToAddrProvider(s.t, s.hdWallet)
	l1Addr := intentbuilder.RoleToAddrProvider(s.t, s.hdWallet, l1ID)
	// The migration runs as the ProxyAdmin owner of every chain, so one key owns all of them.
	proxyAdminOwner := l1Addr(devkeys.SuperchainProxyAdminOwner)
	_, superCfg := builder.WithSuperchain()
	superCfg.WithProxyAdminOwner(proxyAdminOwner).
		WithGuardian(l1Addr(devkeys.SuperchainConfigGuardianKey)).
		WithChallenger(l1Addr(devkeys.ChallengerRole))
	for _, role := range []devkeys.Role{
		devkeys.SuperchainDeployerKey,
		devkeys.SuperchainProxyAdminOwner,
		devkeys.SuperchainConfigGuardianKey,
		devkeys.ChallengerRole,
	} {
		l1Cfg.WithPrefundedAccount(l1Addr(role), *devAccountFunds)
	}
	for i := uint64(0); i < devUserCount; i++ {
		l1Cfg.WithPrefundedAccount(keyAddr(devkeys.ChainUserKeys(l1ID.ToBig())(i)), *devAccountFunds)
	}

	for _, id := range w.L2ChainIDs {
		l2ID := eth.ChainIDFromUInt64(id)
		_, l2Cfg := builder.WithL2(l2ID)
		intentbuilder.WithDevkeyVaults(s.t, s.hdWallet, l2Cfg)
		intentbuilder.WithDevkeyL2Roles(s.t, s.hdWallet, l2Cfg)
		l2Cfg.WithL1ProxyAdminOwner(proxyAdminOwner)
		l2Cfg.WithForkAtGenesis(opforks.Lagoon)

		l2Addr := intentbuilder.RoleToAddrProvider(s.t, s.hdWallet, l2ID)
		for _, role := range []devkeys.Role{
			devkeys.BatcherRole,
			devkeys.DeployerRole,
			devkeys.L1ProxyAdminOwnerRole,
			devkeys.SystemConfigOwner,
			devkeys.ProposerRole,
			devkeys.ChallengerRole,
		} {
			l1Cfg.WithPrefundedAccount(l2Addr(role), *devAccountFunds)
		}
		l2Cfg.WithPrefundedAccount(l2Addr(devkeys.L2ProxyAdminOwnerRole), *devAccountFunds)
		for i := uint64(0); i < devUserCount; i++ {
			l2Cfg.WithPrefundedAccount(keyAddr(devkeys.ChainUserKeys(l2ID.ToBig())(i)), *devAccountFunds)
		}
	}

	intent, err := builder.Build()
	require.NoError(s.t, err)
	intent.FundDevAccounts = true
	return intent
}

// migrateToInterop moves the chains to one dispute game factory that plays SUPER_CANNON_KONA games,
// and returns that factory. The ProxyAdmin owner runs OPContractsManagerV2.migrate on the L1
// through an EIP-7702 delegation.
func (s *interopE2ESystem) migrateToInterop() common.Address {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// The migrator requires the system configs in ascending order of chain ID.
	chains := slices.Clone(s.deployment.Chains)
	slices.SortFunc(chains, func(a, b *state.ChainState) int { return a.ID.Big().Cmp(b.ID.Big()) })
	input := migrateInput{
		DisputeGameConfigs: []disputeGameConfig{{
			Enabled:  true,
			InitBond: superCannonKonaInitBond,
			GameType: uint32(opcm.GameTypeSuperCannonKona),
			GameArgs: konaAbsolutePrestate.Bytes(),
		}},
		// The chains have no super root at genesis, so the anchor is a placeholder.
		StartingAnchorRoot: opcm.Proposal{
			Root:             opcm.PermissionedCannonFallbackPrestatePlaceholder,
			L2SequenceNumber: new(big.Int).SetUint64(s.l1Genesis.Timestamp),
		},
		StartingRespectedGameType: uint32(opcm.GameTypeSuperCannonKona),
	}
	for _, chain := range chains {
		input.ChainSystemConfigs = append(input.ChainSystemConfigs, chain.SystemConfigProxy)
	}
	calldata, err := migrateFn.EncodeArgs(input)
	require.NoError(s.t, err)

	proxyAdminOwner, err := s.hdWallet.Secret(devkeys.SuperchainProxyAdminOwner.Key(s.l1Genesis.Config.ChainID))
	require.NoError(s.t, err)
	client := s.L1GethClient()
	opcmAddr := s.deployment.ImplementationsDeployment.OpcmV2Impl
	require.NoError(s.t, transactions.DelegateCallWithSetCode(ctx, client, proxyAdminOwner, opcmAddr, calldata), "interop migration failed")

	factoryCall, err := disputeGameFactoryFn.EncodeArgs()
	require.NoError(s.t, err)
	var factory common.Address
	for i, chain := range chains {
		portal := chain.OptimismPortalProxy
		out, err := client.CallContract(ctx, ethereum.CallMsg{To: &portal, Data: factoryCall}, nil)
		require.NoError(s.t, err)
		var chainFactory common.Address
		require.NoError(s.t, disputeGameFactoryFn.DecodeReturns(out, &chainFactory))
		if i == 0 {
			factory = chainFactory
		}
		require.Equal(s.t, factory, chainFactory, "chain %s does not use the shared dispute game factory", chain.ID.Big())
	}
	require.NotEqual(s.t, common.Address{}, factory, "the migration left no dispute game factory")
	return factory
}
