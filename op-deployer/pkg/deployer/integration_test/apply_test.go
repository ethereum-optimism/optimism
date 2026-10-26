package integration_test

import (
	"bytes"
	"context"
	"encoding/hex"
	"log/slog"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-chain-ops/addresses"
	"github.com/ethereum-optimism/optimism/op-core/devfeatures"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/bootstrap"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/inspect"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/integration_test/shared"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"

	"github.com/ethereum/go-ethereum"

	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/ethereum-optimism/optimism/op-service/testutils/devnet"

	altda "github.com/ethereum-optimism/optimism/op-alt-da"
	"github.com/ethereum-optimism/optimism/op-node/rollup"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/artifacts"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/pipeline"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/standard"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/testutil"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"

	op_e2e "github.com/ethereum-optimism/optimism/op-e2e"

	"github.com/holiman/uint256"
	"github.com/lmittmann/w3"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/require"
)

const (
	testCustomGasLimit       = uint64(90_123_456)
	testReceiptQueryInterval = 50 * time.Millisecond
)

type deployerKey struct{}

func (d *deployerKey) HDPath() string {
	return "m/44'/60'/0'/0/0"
}

func (d *deployerKey) String() string {
	return "deployer-key"
}

// TestEndToEndBootstrapApply tests that a system can be fully bootstrapped and applied, both from
// local artifacts and the default tagged artifacts. The tagged artifacts test only runs on proposal
// or backports branches, since those are the only branches with an SLA to support tagged artifacts.
func TestEndToEndBootstrapApply(t *testing.T) {
	op_e2e.InitParallel(t)

	lgr := testlog.Logger(t, slog.LevelDebug)
	l1RPC, l1Client := devnet.DefaultAnvilRPC(t, lgr)
	pkHex, pk, dk := shared.DefaultPrivkey(t)
	l1ChainID := new(big.Int).SetUint64(devnet.DefaultChainID)
	l2ChainID := uint256.NewInt(1)
	testCacheDir := testutils.IsolatedTestDirWithAutoCleanup(t)
	superchainPAO := common.Address{'S', 'P', 'A', 'O'}

	apply := func(t *testing.T, loc *artifacts.Locator) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		bstrap, err := bootstrap.Superchain(ctx, bootstrap.SuperchainConfig{
			L1RPCUrl:                  l1RPC,
			PrivateKey:                pkHex,
			Logger:                    lgr,
			ArtifactsLocator:          loc,
			CacheDir:                  testCacheDir,
			SuperchainProxyAdminOwner: superchainPAO,
			Guardian:                  common.Address{'G'},
			Paused:                    false,
		})
		require.NoError(t, err)

		impls, err := bootstrap.Implementations(ctx, bootstrap.ImplementationsConfig{
			L1RPCUrl:                           l1RPC,
			PrivateKey:                         pkHex,
			ArtifactsLocator:                   loc,
			MIPSVersion:                        int(standard.MIPSVersion),
			MinWithdrawalDelaySeconds:          standard.MinWithdrawalDelaySeconds,
			MaxWithdrawalDelaySeconds:          standard.MaxWithdrawalDelaySeconds,
			MinProposalSizeBytes:               standard.MinProposalSizeBytes,
			ChallengePeriodSeconds:             standard.ChallengePeriodSeconds,
			MinProofMaturityDelaySeconds:       standard.MinProofMaturityDelaySeconds,
			MaxProofMaturityDelaySeconds:       standard.MaxProofMaturityDelaySeconds,
			MinDisputeGameFinalityDelaySeconds: standard.MinDisputeGameFinalityDelaySeconds,
			MaxDisputeGameFinalityDelaySeconds: standard.MaxDisputeGameFinalityDelaySeconds,
			DevFeatureBitmap:                   common.Hash{},
			SuperchainConfigProxy:              bstrap.SuperchainConfigProxy,
			L1ProxyAdminOwner:                  superchainPAO,
			SuperchainProxyAdmin:               bstrap.SuperchainProxyAdmin,
			CacheDir:                           testCacheDir,
			Logger:                             lgr,
			Challenger:                         common.Address{'C'},
			FaultGameMaxGameDepth:              standard.DisputeMaxGameDepth,
			FaultGameSplitDepth:                standard.DisputeSplitDepth,
			FaultGameClockExtension:            standard.DisputeClockExtension,
			FaultGameMaxClockDuration:          standard.DisputeMaxClockDuration,
		})
		require.NoError(t, err)

		intent, st := shared.NewIntent(t, l1ChainID, dk, l2ChainID, loc, loc, testCustomGasLimit)
		intent.SuperchainRoles = nil
		intent.OPCMAddress = &impls.OpcmV2
		intent.SuperchainConfigProxy = &bstrap.SuperchainConfigProxy

		require.NoError(t, deployer.ApplyPipeline(
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

		cg := ethClientCodeGetter(ctx, l1Client)
		validateOPChainDeployment(t, cg, st, intent, false)
	}

	t.Run("default tagged artifacts", func(t *testing.T) {
		apply(t, artifacts.DefaultL1ContractsLocator)
	})

	t.Run("local artifacts", func(t *testing.T) {
		loc, _ := testutil.LocalArtifacts(t)
		apply(t, loc)
	})
}

// TestApplyDefaultsSP1VerifierOnSepolia pins that a ZK-enabled live apply with no sp1Verifier
// override deploys an SP1PlonkAdapter wrapping the release-approved raw verifier, and that the
// verifier implements the circuit recorded in standard.SP1VerifierHashFor.
func TestApplyDefaultsSP1VerifierOnSepolia(t *testing.T) {
	op_e2e.InitParallel(t)

	lgr := testlog.Logger(t, slog.LevelDebug)

	forkedL1, stopL1, err := devnet.NewForkedSepolia(lgr)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, stopL1())
	})
	l1RPC := forkedL1.RPCUrl()

	loc, _ := testutil.LocalArtifacts(t)
	testCacheDir := testutils.IsolatedTestDirWithAutoCleanup(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	const sepoliaChainID = 11155111
	_, pk, dk := shared.DefaultPrivkey(t)
	intent, st := shared.NewIntent(t, big.NewInt(sepoliaChainID), dk, uint256.NewInt(12345), loc, loc, 30_000_000)
	intent.GlobalDeployOverrides = map[string]any{
		"devFeatureBitmap": devfeatures.EnableDevFeature(common.Hash{}, devfeatures.ZKDisputeGameFlag),
	}

	require.NoError(t, deployer.ApplyPipeline(ctx, deployer.ApplyPipelineOpts{
		DeploymentTarget:   deployer.DeploymentTargetLive,
		L1RPCUrl:           l1RPC,
		DeployerPrivateKey: pk,
		Intent:             intent,
		State:              st,
		Logger:             lgr,
		StateWriter:        pipeline.NoopStateWriter(),
		CacheDir:           testCacheDir,
	}))

	expected, err := standard.SP1VerifierFor(sepoliaChainID)
	require.NoError(t, err)
	require.NotNil(t, st.SP1Verifier)
	require.Equal(t, expected, *st.SP1Verifier, "apply should persist the selected raw verifier")
	require.NotContains(t, intent.GlobalDeployOverrides, "sp1Verifier", "apply must not rewrite the intent")

	client, err := ethclient.Dial(l1RPC)
	require.NoError(t, err)
	defer client.Close()

	verifierCode, err := client.CodeAt(ctx, expected, nil)
	require.NoError(t, err)
	require.NotEmpty(t, verifierCode, "release verifier must already be deployed on Sepolia")

	adapter := st.ImplementationsDeployment.SP1PlonkAdapterImpl
	require.NotEqual(t, common.Address{}, adapter, "ZK-enabled apply should deploy an SP1PlonkAdapter")

	sp1VerifierFn := w3.MustNewFunc("sp1Verifier()", "address")
	calldata, err := sp1VerifierFn.EncodeArgs()
	require.NoError(t, err)
	ret, err := client.CallContract(ctx, ethereum.CallMsg{To: &adapter, Data: calldata}, nil)
	require.NoError(t, err)
	var wrapped common.Address
	require.NoError(t, sp1VerifierFn.DecodeReturns(ret, &wrapped))
	require.Equal(t, expected, wrapped, "adapter should wrap the release verifier")

	verifierHashFn := w3.MustNewFunc("VERIFIER_HASH()", "bytes32")
	calldata, err = verifierHashFn.EncodeArgs()
	require.NoError(t, err)
	ret, err = client.CallContract(ctx, ethereum.CallMsg{To: &wrapped, Data: calldata}, nil)
	require.NoError(t, err)
	var verifierHash common.Hash
	require.NoError(t, verifierHashFn.DecodeReturns(ret, &verifierHash))
	expectedHash, err := standard.SP1VerifierHashFor(sepoliaChainID)
	require.NoError(t, err)
	require.Equal(t, expectedHash, verifierHash, "release verifier must implement the pinned circuit")
}

func TestEndToEndApply(t *testing.T) {
	op_e2e.InitParallel(t)

	lgr := testlog.Logger(t, slog.LevelDebug)
	l1RPC, l1Client := devnet.DefaultAnvilRPC(t, lgr)
	_, pk, dk := shared.DefaultPrivkey(t)
	l1ChainID := new(big.Int).SetUint64(devnet.DefaultChainID)
	l2ChainID1 := uint256.NewInt(1)
	l2ChainID2 := uint256.NewInt(2)
	loc, _ := testutil.LocalArtifacts(t)
	testCacheDir := testutils.IsolatedTestDirWithAutoCleanup(t)

	t.Run("two chains one after another", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), 5*time.Minute)
		defer cancel()

		intent, st := shared.NewIntent(t, l1ChainID, dk, l2ChainID1, loc, loc, testCustomGasLimit)
		cg := ethClientCodeGetter(ctx, l1Client)

		require.NoError(t, deployer.ApplyPipeline(
			ctx,
			deployer.ApplyPipelineOpts{
				DeploymentTarget:     deployer.DeploymentTargetLive,
				L1RPCUrl:             l1RPC,
				DeployerPrivateKey:   pk,
				Intent:               intent,
				State:                st,
				Logger:               lgr,
				StateWriter:          pipeline.NoopStateWriter(),
				CacheDir:             testCacheDir,
				ReceiptQueryInterval: testReceiptQueryInterval,
			},
		))

		// create a new environment with wiped state to ensure we can continue using the
		// state from the previous deployment
		intent.Chains = append(intent.Chains, shared.NewChainIntent(t, dk, l1ChainID, l2ChainID2, testCustomGasLimit))

		require.NoError(t, deployer.ApplyPipeline(
			ctx,
			deployer.ApplyPipelineOpts{
				DeploymentTarget:     deployer.DeploymentTargetLive,
				L1RPCUrl:             l1RPC,
				DeployerPrivateKey:   pk,
				Intent:               intent,
				State:                st,
				Logger:               lgr,
				StateWriter:          pipeline.NoopStateWriter(),
				CacheDir:             testCacheDir,
				ReceiptQueryInterval: testReceiptQueryInterval,
			},
		))

		validateSuperchainDeployment(t, st, cg, true)
		validateOPChainDeployment(t, cg, st, intent, false)
	})

	t.Run("on the Go script host", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), 5*time.Minute)
		defer cancel()

		// --use-forge=false keeps the Go script host until a later release removes it.
		intent, st := shared.NewIntent(t, l1ChainID, dk, l2ChainID1, loc, loc, testCustomGasLimit)
		cg := ethClientCodeGetter(ctx, l1Client)
		require.NoError(t, deployer.ApplyPipeline(ctx, deployer.ApplyPipelineOpts{
			DeploymentTarget:     deployer.DeploymentTargetLive,
			L1RPCUrl:             l1RPC,
			DeployerPrivateKey:   pk,
			Intent:               intent,
			State:                st,
			Logger:               lgr,
			StateWriter:          pipeline.NoopStateWriter(),
			CacheDir:             testCacheDir,
			ReceiptQueryInterval: testReceiptQueryInterval,
			UseScriptHost:        true,
		}))
		validateSuperchainDeployment(t, st, cg, true)
		validateOPChainDeployment(t, cg, st, intent, false)
	})

	t.Run("with alt-da and additional dispute games", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), 5*time.Minute)
		defer cancel()

		intent, st := shared.NewIntent(t, l1ChainID, dk, l2ChainID1, loc, loc, testCustomGasLimit)
		// The deployer registers the games, so it must own the factory and be the guardian.
		deployerAddr := crypto.PubkeyToAddress(pk.PublicKey)
		intent.Chains[0].Roles.L1ProxyAdminOwner = deployerAddr
		intent.SuperchainRoles.SuperchainGuardian = deployerAddr
		intent.GlobalDeployOverrides = map[string]any{"preimageOracleChallengePeriod": 1}
		intent.Chains[0].DangerousAltDAConfig = genesis.AltDADeployConfig{
			UseAltDA:                   true,
			DACommitmentType:           altda.KeccakCommitmentString,
			DAChallengeWindow:          10,
			DAResolveWindow:            10,
			DABondSize:                 100,
			DAResolverRefundPercentage: 50,
		}
		gameParams := state.ChainProofParams{
			DisputeAbsolutePrestate:                 standard.DisputeAbsolutePrestate,
			DisputeMaxGameDepth:                     50,
			DisputeSplitDepth:                       14,
			DisputeClockExtension:                   1,
			DisputeMaxClockDuration:                 10,
			DangerouslyAllowCustomDisputeParameters: true,
		}
		alphabet, cannon := gameParams, gameParams
		alphabet.DisputeGameType = 255
		cannon.DisputeGameType = 254
		intent.Chains[0].AdditionalDisputeGames = []state.AdditionalDisputeGame{
			{ChainProofParams: alphabet, VMType: state.VMTypeAlphabet, MakeRespected: true},
			{ChainProofParams: cannon, VMType: state.VMTypeCannon},
		}

		require.NoError(t, deployer.ApplyPipeline(ctx, deployer.ApplyPipelineOpts{
			DeploymentTarget:     deployer.DeploymentTargetLive,
			L1RPCUrl:             l1RPC,
			DeployerPrivateKey:   pk,
			Intent:               intent,
			State:                st,
			Logger:               lgr,
			StateWriter:          pipeline.NoopStateWriter(),
			CacheDir:             testCacheDir,
			ReceiptQueryInterval: testReceiptQueryInterval,
		}))

		chainState := st.Chains[0]
		require.NotEmpty(t, chainState.AltDAChallengeProxy)
		require.NotEmpty(t, chainState.AltDAChallengeImpl)
		require.Len(t, chainState.AdditionalDisputeGames, 2)
		gameImpls := w3.MustNewFunc("gameImpls(uint32)", "address")
		for _, game := range chainState.AdditionalDisputeGames {
			calldata, err := gameImpls.EncodeArgs(game.GameType)
			require.NoError(t, err)
			ret, err := l1Client.CallContract(ctx, ethereum.CallMsg{To: &chainState.DisputeGameFactoryProxy, Data: calldata}, nil)
			require.NoError(t, err)
			var impl common.Address
			require.NoError(t, gameImpls.DecodeReturns(ret, &impl))
			require.Equal(t, game.GameAddress, impl, "game type %d is not registered on L1", game.GameType)
		}
	})

	t.Run("with custom gas token", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), 5*time.Minute)
		defer cancel()

		intent, st := shared.NewIntent(t, l1ChainID, dk, l2ChainID1, loc, loc, testCustomGasLimit)

		// CGT config for L2 genesis
		amount := new(big.Int)
		amount.SetString("1000000000000000000000", 10)
		intent.Chains[0].CustomGasToken = state.CustomGasToken{
			Name:             "Custom Gas Token",
			Symbol:           "CGT",
			InitialLiquidity: (*hexutil.Big)(amount),
		}

		require.NoError(t, deployer.ApplyPipeline(ctx, deployer.ApplyPipelineOpts{
			DeploymentTarget:     deployer.DeploymentTargetLive,
			L1RPCUrl:             l1RPC,
			DeployerPrivateKey:   pk,
			Intent:               intent,
			State:                st,
			Logger:               lgr,
			StateWriter:          pipeline.NoopStateWriter(),
			CacheDir:             testCacheDir,
			ReceiptQueryInterval: testReceiptQueryInterval,
		}))

		systemConfig := st.Chains[0].SystemConfigProxy
		fn := w3.MustNewFunc("isFeatureEnabled(bytes32)", "bool")
		// bytes32("CUSTOM_GAS_TOKEN")
		data, err := fn.EncodeArgs(w3.H("0x435553544f4d5f4741535f544f4b454e00000000000000000000000000000000"))
		require.NoError(t, err)

		res, err := l1Client.CallContract(ctx, ethereum.CallMsg{
			To:   &systemConfig,
			Data: data,
		}, nil)
		require.NoError(t, err)

		var response bool
		err = fn.DecodeReturns(res, &response)
		require.NoError(t, err)
		require.Equal(t, true, response)

		// Check that the native asset liquidity predeploy has the configured amount in L2 genesis
		nativeAssetLiquidityAddr := common.HexToAddress("0x4200000000000000000000000000000000000029")
		l2Genesis := st.Chains[0].Allocs.Data.Accounts
		account, exists := l2Genesis[nativeAssetLiquidityAddr]
		require.True(t, exists, "Native asset liquidity predeploy should exist in L2 genesis")
		require.Equal(t, amount, account.Balance, "Native asset liquidity predeploy should have the configured balance")
	})

	t.Run("OPCMV2 deployment", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		lgr := testlog.Logger(t, slog.LevelDebug)
		l1RPC, l1Client := devnet.DefaultAnvilRPC(t, lgr)
		_, pk, dk := shared.DefaultPrivkey(t)
		l1ChainID := new(big.Int).SetUint64(devnet.DefaultChainID)
		l2ChainID := uint256.NewInt(1)
		loc, _ := testutil.LocalArtifacts(t)
		testCacheDir := testutils.IsolatedTestDirWithAutoCleanup(t)

		intent, st := shared.NewIntent(t, l1ChainID, dk, l2ChainID, loc, loc, testCustomGasLimit)

		require.NoError(t, deployer.ApplyPipeline(
			ctx,
			deployer.ApplyPipelineOpts{
				DeploymentTarget:     deployer.DeploymentTargetLive,
				L1RPCUrl:             l1RPC,
				DeployerPrivateKey:   pk,
				Intent:               intent,
				State:                st,
				Logger:               lgr,
				StateWriter:          pipeline.NoopStateWriter(),
				CacheDir:             testCacheDir,
				ReceiptQueryInterval: testReceiptQueryInterval,
			},
		))

		// Verify that OPCMV2 was deployed in implementations
		require.NotEmpty(t, st.ImplementationsDeployment.OpcmV2Impl, "OPCMV2 implementation should be deployed")
		require.NotEmpty(t, st.ImplementationsDeployment.OpcmContainerImpl, "OPCM container implementation should be deployed")
		require.NotEmpty(t, st.ImplementationsDeployment.OpcmStandardValidatorImpl, "OPCM standard validator implementation should be deployed")

		// Verify that implementations are deployed on L1
		cg := ethClientCodeGetter(ctx, l1Client)

		opcmV2Code := cg(t, st.ImplementationsDeployment.OpcmV2Impl)
		require.NotEmpty(t, opcmV2Code, "OPCMV2 should have code deployed")

		require.NotEqual(t, common.Address{}, st.ImplementationsDeployment.OpcmV2Impl, "OpcmV2Impl should be set")
	})
}

func TestGlobalOverrides(t *testing.T) {
	op_e2e.InitParallel(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	opts, intent, st := setupGenesisChain(t, devnet.DefaultChainID)
	expectedBaseFeeVaultRecipient := common.HexToAddress("0x0000000000000000000000000000000000000001")
	expectedL1FeeVaultRecipient := common.HexToAddress("0x0000000000000000000000000000000000000002")
	expectedSequencerFeeVaultRecipient := common.HexToAddress("0x0000000000000000000000000000000000000003")
	expectedOperatorFeeVaultRecipient := common.HexToAddress("0x0000000000000000000000000000000000000004")
	expectedBaseFeeVaultMinimumWithdrawalAmount := strings.ToLower("0x1BC16D674EC80000")
	expectedBaseFeeVaultWithdrawalNetwork := genesis.FromUint8(0)
	expectedEnableGovernance := false
	expectedGasPriceOracleBaseFeeScalar := uint32(1300)
	expectedEIP1559Denominator := uint64(500)
	expectedUseFaultProofs := false
	intent.GlobalDeployOverrides = map[string]interface{}{
		"l2BlockTime":                         float64(3),
		"baseFeeVaultRecipient":               expectedBaseFeeVaultRecipient,
		"l1FeeVaultRecipient":                 expectedL1FeeVaultRecipient,
		"sequencerFeeVaultRecipient":          expectedSequencerFeeVaultRecipient,
		"operatorFeeVaultRecipient":           expectedOperatorFeeVaultRecipient,
		"baseFeeVaultMinimumWithdrawalAmount": expectedBaseFeeVaultMinimumWithdrawalAmount,
		"baseFeeVaultWithdrawalNetwork":       expectedBaseFeeVaultWithdrawalNetwork,
		"enableGovernance":                    expectedEnableGovernance,
		"gasPriceOracleBaseFeeScalar":         expectedGasPriceOracleBaseFeeScalar,
		"eip1559Denominator":                  expectedEIP1559Denominator,
		"useFaultProofs":                      expectedUseFaultProofs,
	}

	require.NoError(t, deployer.ApplyPipeline(ctx, opts))

	cfg, err := state.CombineDeployConfig(intent, intent.Chains[0], st, st.Chains[0])
	require.NoError(t, err)
	require.Equal(t, uint64(3), cfg.L2InitializationConfig.L2CoreDeployConfig.L2BlockTime, "L2 block time should be 3 seconds")
	require.Equal(t, expectedBaseFeeVaultRecipient, cfg.L2InitializationConfig.L2VaultsDeployConfig.BaseFeeVaultRecipient, "Base Fee Vault Recipient should be the expected address")
	require.Equal(t, expectedL1FeeVaultRecipient, cfg.L2InitializationConfig.L2VaultsDeployConfig.L1FeeVaultRecipient, "L1 Fee Vault Recipient should be the expected address")
	require.Equal(t, expectedSequencerFeeVaultRecipient, cfg.L2InitializationConfig.L2VaultsDeployConfig.SequencerFeeVaultRecipient, "Sequencer Fee Vault Recipient should be the expected address")
	require.Equal(t, expectedBaseFeeVaultMinimumWithdrawalAmount, strings.ToLower(cfg.L2InitializationConfig.L2VaultsDeployConfig.BaseFeeVaultMinimumWithdrawalAmount.String()), "Base Fee Vault Minimum Withdrawal Amount should be the expected value")
	require.Equal(t, expectedBaseFeeVaultWithdrawalNetwork, cfg.L2InitializationConfig.L2VaultsDeployConfig.BaseFeeVaultWithdrawalNetwork, "Base Fee Vault Withdrawal Network should be the expected value")
	require.Equal(t, expectedEnableGovernance, cfg.L2InitializationConfig.GovernanceDeployConfig.EnableGovernance, "Governance should be disabled")
	require.Equal(t, expectedGasPriceOracleBaseFeeScalar, cfg.L2InitializationConfig.GasPriceOracleDeployConfig.GasPriceOracleBaseFeeScalar, "Gas Price Oracle Base Fee Scalar should be the expected value")
	require.Equal(t, expectedEIP1559Denominator, cfg.L2InitializationConfig.EIP1559DeployConfig.EIP1559Denominator, "EIP-1559 Denominator should be the expected value")
	require.Equal(t, expectedUseFaultProofs, cfg.UseFaultProofs, "Fault proofs should not be enabled")
}

func TestApplyGenesisStrategy(t *testing.T) {
	op_e2e.InitParallel(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pragueOffset := uint64(2000)
	l1GenesisParams := &state.L1DevGenesisParams{
		BlockParams: state.L1DevGenesisBlockParams{
			Timestamp:     1000,
			GasLimit:      42_000_000,
			ExcessBlobGas: 9000,
		},
		PragueTimeOffset: &pragueOffset,
	}

	deployChain := func(l1DevGenesisParams *state.L1DevGenesisParams) *state.State {
		opts, intent, st := setupGenesisChain(t, devnet.DefaultChainID)
		intent.L1DevGenesisParams = l1DevGenesisParams
		require.NoError(t, deployer.ApplyPipeline(ctx, opts))
		cg := stateDumpCodeGetter(st)
		validateSuperchainDeployment(t, st, cg, true)
		validateOPChainDeployment(t, cg, st, intent, false)
		return st
	}

	t.Run("defaults", func(t *testing.T) {
		st := deployChain(nil)
		require.Greater(t, st.Chains[0].StartBlock.Time, l1GenesisParams.BlockParams.Timestamp)
		require.Nil(t, st.L1DevGenesis.Config.PragueTime)
	})

	t.Run("custom", func(t *testing.T) {
		st := deployChain(l1GenesisParams)
		require.EqualValues(t, l1GenesisParams.BlockParams.Timestamp, st.Chains[0].StartBlock.Time)
		require.EqualValues(t, l1GenesisParams.BlockParams.Timestamp, st.L1DevGenesis.Timestamp)

		require.EqualValues(t, l1GenesisParams.BlockParams.GasLimit, st.L1DevGenesis.GasLimit)
		require.NotNil(t, st.L1DevGenesis.ExcessBlobGas)
		require.EqualValues(t, l1GenesisParams.BlockParams.ExcessBlobGas, *st.L1DevGenesis.ExcessBlobGas)
		require.NotNil(t, st.L1DevGenesis.Config.PragueTime)
		expectedPragueTimestamp := l1GenesisParams.BlockParams.Timestamp + *l1GenesisParams.PragueTimeOffset
		require.EqualValues(t, expectedPragueTimestamp, *st.L1DevGenesis.Config.PragueTime)
	})
}

func TestContinuationDeploymentUsesPreparedInputs(t *testing.T) {
	op_e2e.InitParallel(t)

	ctx := t.Context()

	opts, _, phaseOneState := setupGenesisChain(t, devnet.DefaultChainID)
	require.NoError(t, deployer.ApplyPipeline(ctx, opts))
	require.NotNil(t, phaseOneState.ImplementationsDeployment)
	require.NotNil(t, phaseOneState.SuperchainDeployment)
	require.NotNil(t, phaseOneState.L1StateDump)

	opcmAddress := phaseOneState.ImplementationsDeployment.OpcmV2Impl
	superchainConfigProxy := phaseOneState.SuperchainDeployment.SuperchainConfigProxy
	l1StateDump := phaseOneState.L1StateDump.Data
	deployerAddress := crypto.PubkeyToAddress(opts.DeployerPrivateKey.PublicKey)
	require.NotZero(t, phaseOneState.Create2Salt)
	require.NotZero(t, opcmAddress)
	require.NotZero(t, superchainConfigProxy)
	require.NotEqual(t, standard.PlaceholderAddress, deployerAddress)

	loc, l1Artifacts := testutil.LocalArtifacts(t)
	dk, err := devkeys.NewMnemonicDevKeys(devkeys.TestMnemonic)
	require.NoError(t, err)
	l1ChainID := new(big.Int).SetUint64(devnet.DefaultChainID)
	committedPrestate := common.HexToHash("0x123456")
	committedAnchor := &state.StartingAnchorProposal{
		Root:             common.HexToHash("0xabcdef"),
		L2SequenceNumber: 7,
	}

	newContinuationInputs := func(t *testing.T, l2ChainID *uint256.Int) (*state.Intent, *state.State, common.Hash) {
		t.Helper()

		intent, _ := shared.NewIntent(t, l1ChainID, dk, l2ChainID, loc, loc, testCustomGasLimit)
		intent.SuperchainConfigProxy = &superchainConfigProxy
		intent.Chains[0].DeployOverrides = map[string]any{
			"respectedGameType": opcm.GameTypeSuperCannonKona,
		}

		st := &state.State{
			Version:     1,
			Create2Salt: phaseOneState.Create2Salt,
			PreparedDeployment: &state.PreparedDeployment{
				Intent:   intent,
				Deployer: deployerAddress,
				OPCM:     opcmAddress,
			},
		}
		chainID := intent.Chains[0].ID
		// Deployment does not read predicted chain addresses.
		st.SetChainContracts(chainID, addresses.OpChainContracts{}, false)
		chainState, err := st.Chain(chainID)
		require.NoError(t, err)
		gameType := uint32(opcm.GameTypeSuperCannonKona)
		chainState.InitialGameType = &gameType
		chainState.Prestate = committedPrestate
		chainState.StartingAnchorRoot = committedAnchor
		st.PreparedDeployment.Chains = []*state.PreparedChainState{{ID: chainID}}
		return intent, st, chainID
	}

	_, st, chainID := newContinuationInputs(t, uint256.NewInt(2))
	require.Nil(t, st.ImplementationsDeployment)
	require.Nil(t, st.SuperchainDeployment)
	// Continue on an L1 that holds the state of phase one.
	l1RPCURL, l1Client := devnet.DefaultAnvilRPC(t, opts.Logger)
	shared.LoadAllocs(t, l1Client.Client(), l1StateDump)
	require.NoError(t, l1Client.Client().Call(nil, "anvil_setBalance", deployerAddress, (*hexutil.Big)(big.NewInt(params.Ether))))
	forgeClient, err := opcm.NewForgeClient(l1Artifacts, "", opcm.L1ForgeScripts)
	require.NoError(t, err)
	pEnv := &pipeline.Env{
		StateWriter: pipeline.NoopStateWriter(),
		L1Client:    l1Client,
		Deployer:    deployerAddress,
		Logger:      opts.Logger,
		Forge: &opcm.ForgeEnv{
			Client:     forgeClient,
			Context:    ctx,
			L1RPCUrl:   l1RPCURL,
			PrivateKey: hexutil.Encode(crypto.FromECDSA(opts.DeployerPrivateKey)),
		},
		Context: ctx,
	}

	dci, err := pipeline.BuildContinuationDCI(chainID, st)
	require.NoError(t, err)
	result, err := pipeline.ExecuteOPChainDeployment(pEnv, st, chainID, dci)
	require.NoError(t, err)

	chainState, err := st.Chain(chainID)
	require.NoError(t, err)
	require.False(t, *chainState.Deployed)
	require.NoError(t, pipeline.RecordOPChainDeployment(st, result))
	chainState, err = st.Chain(chainID)
	require.NoError(t, err)
	require.True(t, *chainState.Deployed)

	call := func(to common.Address, data []byte) ([]byte, error) {
		return l1Client.CallContract(ctx, ethereum.CallMsg{To: &to, Data: data}, nil)
	}
	gameImpls := w3.MustNewFunc("gameImpls(uint32)", "address")
	readGameImpl := func(gameType opcm.GameType) common.Address {
		callData, err := gameImpls.EncodeArgs(uint32(gameType))
		require.NoError(t, err)
		ret, err := call(chainState.DisputeGameFactoryProxy, callData)
		require.NoError(t, err)
		var implementation common.Address
		require.NoError(t, gameImpls.DecodeReturns(ret, &implementation))
		require.NotZero(t, implementation)
		return implementation
	}
	superCannonKonaImpl := readGameImpl(opcm.GameTypeSuperCannonKona)
	superPermissionedImpl := readGameImpl(opcm.GameTypeSuperPermissioned)
	require.NotEqual(t, superCannonKonaImpl, superPermissionedImpl)

	gameArgs := w3.MustNewFunc("gameArgs(uint32)", "bytes")
	readAbsolutePrestate := func(gameType opcm.GameType) common.Hash {
		callData, err := gameArgs.EncodeArgs(uint32(gameType))
		require.NoError(t, err)
		ret, err := call(chainState.DisputeGameFactoryProxy, callData)
		require.NoError(t, err)
		var args []byte
		require.NoError(t, gameArgs.DecodeReturns(ret, &args))
		require.GreaterOrEqual(t, len(args), common.HashLength)
		return common.BytesToHash(args[:common.HashLength])
	}
	require.Equal(t, committedPrestate, readAbsolutePrestate(opcm.GameTypeSuperCannonKona))

	getStartingAnchorRoot := w3.MustNewFunc(
		"getStartingAnchorRoot()",
		"(bytes32 root, uint256 l2SequenceNumber)",
	)
	callData, err := getStartingAnchorRoot.EncodeArgs()
	require.NoError(t, err)
	ret, err := call(chainState.AnchorStateRegistryProxy, callData)
	require.NoError(t, err)
	var actualAnchor opcm.Proposal
	require.NoError(t, getStartingAnchorRoot.DecodeReturns(ret, &actualAnchor))
	require.Equal(t, committedAnchor.Root, actualAnchor.Root)
	require.Equal(
		t,
		new(big.Int).SetUint64(uint64(committedAnchor.L2SequenceNumber)),
		actualAnchor.L2SequenceNumber,
	)

	respectedGameType := w3.MustNewFunc("respectedGameType()", "uint32")
	callData, err = respectedGameType.EncodeArgs()
	require.NoError(t, err)
	ret, err = call(chainState.AnchorStateRegistryProxy, callData)
	require.NoError(t, err)
	var actualGameType uint32
	require.NoError(t, respectedGameType.DecodeReturns(ret, &actualGameType))
	require.Equal(t, uint32(opcm.GameTypeSuperCannonKona), actualGameType)

	t.Run("missing committed prestate", func(t *testing.T) {
		_, st, chainID := newContinuationInputs(t, uint256.NewInt(3))
		chainState, err := st.Chain(chainID)
		require.NoError(t, err)
		chainState.Prestate = common.Hash{}

		_, err = pipeline.BuildContinuationDCI(chainID, st)
		require.ErrorContains(t, err, "op-deployer prestate")
		require.False(t, *chainState.Deployed)
	})
}

func TestProofParamOverrides(t *testing.T) {
	op_e2e.InitParallel(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	opts, intent, st := setupGenesisChain(t, devnet.DefaultChainID)
	intent.GlobalDeployOverrides = map[string]any{
		"preimageOracleMinProposalSize": standard.MinProposalSizeBytes + 1,
		"preimageOracleChallengePeriod": standard.ChallengePeriodSeconds + 1,
		// Implementation bounds, and per-chain delays that sit inside them.
		"minProofMaturityDelaySeconds":            standard.MinProofMaturityDelaySeconds + 1,
		"maxProofMaturityDelaySeconds":            standard.MaxProofMaturityDelaySeconds + 1,
		"minDisputeGameFinalityDelaySeconds":      standard.MinDisputeGameFinalityDelaySeconds + 1,
		"maxDisputeGameFinalityDelaySeconds":      standard.MaxDisputeGameFinalityDelaySeconds + 1,
		"minWithdrawalDelaySeconds":               standard.MinWithdrawalDelaySeconds + 1,
		"maxWithdrawalDelaySeconds":               standard.MaxWithdrawalDelaySeconds + 1,
		"proofMaturityDelaySeconds":               standard.ProofMaturityDelaySeconds + 1,
		"disputeGameFinalityDelaySeconds":         standard.DisputeGameFinalityDelaySeconds + 1,
		"faultGameWithdrawalDelay":                standard.WithdrawalDelaySeconds + 1,
		"mipsVersion":                             standard.MIPSVersion,     // Contract enforces a valid value be used
		"respectedGameType":                       standard.DisputeGameType, // This must be set to the permissioned game
		"faultGameAbsolutePrestate":               common.Hash{'A', 'B', 'S', 'O', 'L', 'U', 'T', 'E'},
		"faultGameMaxDepth":                       standard.DisputeMaxGameDepth + 1,
		"faultGameSplitDepth":                     standard.DisputeSplitDepth + 1,
		"faultGameClockExtension":                 standard.DisputeClockExtension + 1,
		"faultGameMaxClockDuration":               standard.DisputeMaxClockDuration + 1,
		"dangerouslyAllowCustomDisputeParameters": true,
		"devFeatureBitmap":                        common.Hash{},
	}

	require.NoError(t, deployer.ApplyPipeline(ctx, opts))

	allocs := st.L1StateDump.Data.Accounts

	uint64Caster := func(t *testing.T, val any) common.Hash {
		return common.BigToHash(new(big.Int).SetUint64(val.(uint64)))
	}

	pdgImpl := st.ImplementationsDeployment.PermissionedDisputeGameImpl
	tests := []struct {
		name    string
		caster  func(t *testing.T, val any) common.Hash
		address common.Address
	}{
		{
			"minWithdrawalDelaySeconds",
			uint64Caster,
			st.ImplementationsDeployment.DelayedWethImpl,
		},
		{
			"maxWithdrawalDelaySeconds",
			uint64Caster,
			st.ImplementationsDeployment.DelayedWethImpl,
		},
		{
			"preimageOracleMinProposalSize",
			uint64Caster,
			st.ImplementationsDeployment.PreimageOracleImpl,
		},
		{
			"preimageOracleChallengePeriod",
			uint64Caster,
			st.ImplementationsDeployment.PreimageOracleImpl,
		},
		{
			"minProofMaturityDelaySeconds",
			uint64Caster,
			st.ImplementationsDeployment.OptimismPortalImpl,
		},
		{
			"maxProofMaturityDelaySeconds",
			uint64Caster,
			st.ImplementationsDeployment.OptimismPortalImpl,
		},
		{
			"minDisputeGameFinalityDelaySeconds",
			uint64Caster,
			st.ImplementationsDeployment.AnchorStateRegistryImpl,
		},
		{
			"maxDisputeGameFinalityDelaySeconds",
			uint64Caster,
			st.ImplementationsDeployment.AnchorStateRegistryImpl,
		},
		{
			"faultGameMaxDepth",
			uint64Caster,
			pdgImpl,
		},
		{
			"faultGameSplitDepth",
			uint64Caster,
			pdgImpl,
		},
		{
			"faultGameClockExtension",
			uint64Caster,
			pdgImpl,
		},
		{
			"faultGameMaxClockDuration",
			uint64Caster,
			pdgImpl,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checkImmutable(t, allocs, tt.address, tt.caster(t, intent.GlobalDeployOverrides[tt.name]))
		})
	}

	// The per-chain delays are no longer immutables: they live in proxy storage on the
	// OptimismPortal (slot 64), the AnchorStateRegistry (slot 7) and the DelayedWETH (slot 6).
	chainState := st.Chains[0]
	t.Run("faultGameWithdrawalDelay", func(t *testing.T) {
		expected := uint64Caster(t, intent.GlobalDeployOverrides["faultGameWithdrawalDelay"])
		checkStorageSlot(t, allocs, chainState.DelayedWethPermissionlessGameProxy, common.BigToHash(big.NewInt(6)), expected)
		checkStorageSlot(t, allocs, chainState.DelayedWethPermissionedGameProxy, common.BigToHash(big.NewInt(6)), expected)
	})
	t.Run("proofMaturityDelaySeconds", func(t *testing.T) {
		checkStorageSlot(
			t,
			allocs,
			chainState.OptimismPortalProxy,
			common.BigToHash(big.NewInt(64)),
			uint64Caster(t, intent.GlobalDeployOverrides["proofMaturityDelaySeconds"]),
		)
	})
	t.Run("disputeGameFinalityDelaySeconds", func(t *testing.T) {
		checkStorageSlot(
			t,
			allocs,
			chainState.AnchorStateRegistryProxy,
			common.BigToHash(big.NewInt(7)),
			uint64Caster(t, intent.GlobalDeployOverrides["disputeGameFinalityDelaySeconds"]),
		)
	})
}

func TestAltDADeployment(t *testing.T) {
	op_e2e.InitParallel(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	opts, intent, st := setupGenesisChain(t, devnet.DefaultChainID)
	altDACfg := genesis.AltDADeployConfig{
		UseAltDA:                   true,
		DACommitmentType:           altda.KeccakCommitmentString,
		DAChallengeWindow:          10,
		DAResolveWindow:            10,
		DABondSize:                 100,
		DAResolverRefundPercentage: 50,
	}
	intent.Chains[0].DangerousAltDAConfig = altDACfg

	require.NoError(t, deployer.ApplyPipeline(ctx, opts))

	chainState := st.Chains[0]
	require.NotEmpty(t, chainState.AltDAChallengeProxy)
	require.NotEmpty(t, chainState.AltDAChallengeImpl)

	_, rollupCfg, err := pipeline.RenderGenesisAndRollup(st, chainState.ID, nil)
	require.NoError(t, err)
	require.EqualValues(t, &rollup.AltDAConfig{
		CommitmentType:     altda.KeccakCommitmentString,
		DAChallengeWindow:  altDACfg.DAChallengeWindow,
		DAChallengeAddress: chainState.AltDAChallengeProxy,
		DAResolveWindow:    altDACfg.DAResolveWindow,
	}, rollupCfg.AltDAConfig)
}

func TestInvalidL2Genesis(t *testing.T) {
	op_e2e.InitParallel(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// these tests were generated by grepping all usages of the deploy
	// config in L2Genesis.s.sol.
	tests := []struct {
		name      string
		overrides map[string]any
	}{
		{
			name: "L2 proxy admin owner not set",
			overrides: map[string]any{
				"proxyAdminOwner": nil,
			},
		},
		{
			name: "base fee vault recipient not set",
			overrides: map[string]any{
				"baseFeeVaultRecipient": nil,
			},
		},
		{
			name: "l1 fee vault recipient not set",
			overrides: map[string]any{
				"l1FeeVaultRecipient": nil,
			},
		},
		{
			name: "sequencer fee vault recipient not set",
			overrides: map[string]any{
				"sequencerFeeVaultRecipient": nil,
			},
		},
		{
			name: "operator fee vault recipient not set",
			overrides: map[string]any{
				"operatorFeeVaultRecipient": nil,
			},
		},
		{
			name: "l1 chain ID not set",
			overrides: map[string]any{
				"l1ChainID": nil,
			},
		},
		{
			name: "l2 chain ID not set",
			overrides: map[string]any{
				"l2ChainID": nil,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts, intent, _ := setupGenesisChain(t, devnet.DefaultChainID)
			intent.GlobalDeployOverrides = tt.overrides

			err := deployer.ApplyPipeline(ctx, opts)
			require.Error(t, err)
			require.ErrorContains(t, err, "failed to combine L2 init config")
		})
	}
}

func TestAdditionalDisputeGames(t *testing.T) {
	op_e2e.InitParallel(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	opts, intent, st := setupGenesisChain(t, devnet.DefaultChainID)
	deployerAddr := crypto.PubkeyToAddress(opts.DeployerPrivateKey.PublicKey)
	(&intent.Chains[0].Roles).L1ProxyAdminOwner = deployerAddr
	intent.SuperchainRoles.SuperchainGuardian = deployerAddr
	intent.GlobalDeployOverrides = map[string]any{
		"preimageOracleChallengePeriod": 1,
	}
	intent.Chains[0].AdditionalDisputeGames = []state.AdditionalDisputeGame{
		{
			ChainProofParams: state.ChainProofParams{
				DisputeGameType:                         255,
				DisputeAbsolutePrestate:                 standard.DisputeAbsolutePrestate,
				DisputeMaxGameDepth:                     50,
				DisputeSplitDepth:                       14,
				DisputeClockExtension:                   1,
				DisputeMaxClockDuration:                 10,
				DangerouslyAllowCustomDisputeParameters: true,
			},
			MakeRespected: true,
			VMType:        state.VMTypeAlphabet,
		},
	}

	require.NoError(t, deployer.ApplyPipeline(ctx, opts))

	chainState := st.Chains[0]
	require.Equal(t, 1, len(chainState.AdditionalDisputeGames))

	gameInfo := chainState.AdditionalDisputeGames[0]
	require.NotEmpty(t, gameInfo.VMAddress)
	require.NotEmpty(t, gameInfo.GameAddress)
	require.NotEmpty(t, gameInfo.OracleAddress)
	require.Equal(t, st.ImplementationsDeployment.PreimageOracleImpl, gameInfo.OracleAddress)
}

func TestIntentConfiguration(t *testing.T) {
	op_e2e.InitParallel(t)

	tests := []struct {
		name       string
		mutator    func(*state.Intent)
		assertions func(t *testing.T, st *state.State)
	}{
		{
			"governance token disabled by default",
			func(intent *state.Intent) {},
			func(t *testing.T, st *state.State) {
				l2Genesis := st.Chains[0].Allocs.Data
				_, ok := l2Genesis.Accounts[predeploys.GovernanceTokenAddr]
				require.False(t, ok)
			},
		},
		{
			"governance token enabled via override",
			func(intent *state.Intent) {
				intent.GlobalDeployOverrides = map[string]any{
					"enableGovernance":     true,
					"governanceTokenOwner": common.Address{'O'}.Hex(),
				}
			},
			func(t *testing.T, st *state.State) {
				l2Genesis := st.Chains[0].Allocs.Data
				_, ok := l2Genesis.Accounts[predeploys.GovernanceTokenAddr]
				require.True(t, ok)
				checkStorageSlot(
					t,
					l2Genesis.Accounts,
					predeploys.GovernanceTokenAddr,
					common.Hash{31: 0x0a},
					common.BytesToHash(common.Address{'O'}.Bytes()),
				)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			opts, intent, st := setupGenesisChain(t, devnet.DefaultChainID)
			tt.mutator(intent)
			require.NoError(t, deployer.ApplyPipeline(ctx, opts))
			tt.assertions(t, st)
		})
	}
}

func setupGenesisChain(t *testing.T, l1ChainID uint64) (deployer.ApplyPipelineOpts, *state.Intent, *state.State) {
	lgr := testlog.Logger(t, slog.LevelDebug)

	depKey := new(deployerKey)
	l1ChainIDBig := new(big.Int).SetUint64(l1ChainID)
	dk, err := devkeys.NewMnemonicDevKeys(devkeys.TestMnemonic)
	require.NoError(t, err)

	l2ChainID1 := uint256.NewInt(1)

	priv, err := dk.Secret(depKey)
	require.NoError(t, err)

	loc, _ := testutil.LocalArtifacts(t)

	intent, st := shared.NewIntent(t, l1ChainIDBig, dk, l2ChainID1, loc, loc, testCustomGasLimit)

	testCacheDir := testutils.IsolatedTestDirWithAutoCleanup(t)

	opts := deployer.ApplyPipelineOpts{
		DeploymentTarget:   deployer.DeploymentTargetGenesis,
		DeployerPrivateKey: priv,
		Intent:             intent,
		State:              st,
		Logger:             lgr,
		StateWriter:        pipeline.NoopStateWriter(),
		CacheDir:           testCacheDir,
	}

	return opts, intent, st
}

type codeGetter func(t *testing.T, addr common.Address) []byte

func ethClientCodeGetter(ctx context.Context, client *ethclient.Client) codeGetter {
	return func(t *testing.T, addr common.Address) []byte {
		code, err := client.CodeAt(ctx, addr, nil)
		require.NoError(t, err)
		return code
	}
}

func stateDumpCodeGetter(st *state.State) codeGetter {
	return func(t *testing.T, addr common.Address) []byte {
		acc, ok := st.L1StateDump.Data.Accounts[addr]
		require.True(t, ok, "no account found for address %s", addr)
		return acc.Code
	}
}

func validateSuperchainDeployment(t *testing.T, st *state.State, cg codeGetter, includeSuperchainImpls bool) {
	type addrTuple struct {
		name string
		addr common.Address
	}
	addrs := []addrTuple{
		{"SuperchainProxyAdminImpl", st.SuperchainDeployment.SuperchainProxyAdminImpl},
		{"SuperchainConfigProxy", st.SuperchainDeployment.SuperchainConfigProxy},
		{"OpcmV2Impl", st.ImplementationsDeployment.OpcmV2Impl},
		{"PreimageOracleImpl", st.ImplementationsDeployment.PreimageOracleImpl},
		{"MipsImpl", st.ImplementationsDeployment.MipsImpl},
	}

	if includeSuperchainImpls {
		addrs = append(addrs, addrTuple{"SuperchainConfigImpl", st.SuperchainDeployment.SuperchainConfigImpl})
	}

	for _, addr := range addrs {
		t.Run(addr.name, func(t *testing.T) {
			code := cg(t, addr.addr)
			require.NotEmpty(t, code, "contract %s at %s has no code", addr.name, addr.addr)
		})
	}
}

func validateOPChainDeployment(t *testing.T, cg codeGetter, st *state.State, intent *state.Intent, govEnabled bool) {
	// Validate that the implementation addresses are always set, even in subsequent deployments
	// that pull from an existing OPCM deployment.
	type addrTuple struct {
		name string
		addr common.Address
	}
	implAddrs := []addrTuple{
		{"DelayedWethImpl", st.ImplementationsDeployment.DelayedWethImpl},
		{"OptimismPortalImpl", st.ImplementationsDeployment.OptimismPortalImpl},
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

	for i, chainState := range st.Chains {
		chainAddrs := []struct {
			name string
			addr common.Address
		}{
			{"ProxyAdminAddress", chainState.OpChainContracts.OpChainProxyAdminImpl},
			{"AddressManagerAddress", chainState.OpChainContracts.AddressManagerImpl},
			{"L1ERC721BridgeProxyAddress", chainState.OpChainContracts.L1Erc721BridgeProxy},
			{"SystemConfigProxyAddress", chainState.OpChainContracts.SystemConfigProxy},
			{"OptimismMintableERC20FactoryProxyAddress", chainState.OpChainContracts.OptimismMintableErc20FactoryProxy},
			{"L1StandardBridgeProxyAddress", chainState.OpChainContracts.L1StandardBridgeProxy},
			{"L1CrossDomainMessengerProxyAddress", chainState.OpChainContracts.L1CrossDomainMessengerProxy},
			{"OptimismPortalProxyAddress", chainState.OpChainContracts.OptimismPortalProxy},
			{"DisputeGameFactoryProxyAddress", chainState.DisputeGameFactoryProxy},
			{"AnchorStateRegistryProxyAddress", chainState.OpChainContracts.AnchorStateRegistryProxy},
			{"FaultDisputeGameAddress", chainState.OpChainContracts.FaultDisputeGameImpl},
			{"PermissionedDisputeGameAddress", chainState.OpChainContracts.PermissionedDisputeGameImpl},
			{"DelayedWETHPermissionedGameProxyAddress", chainState.OpChainContracts.DelayedWethPermissionedGameProxy},
			// {"DelayedWETHPermissionlessGameProxyAddress", chainState.DelayedWETHPermissionlessGameProxyAddress},
		}
		for _, addr := range chainAddrs {
			// TODO Delete this `if`` block once FaultDisputeGameAddress is deployed.
			if addr.name == "FaultDisputeGameAddress" {
				continue
			}
			code := cg(t, addr.addr)
			require.NotEmpty(t, code, "contract %s at %s for chain %s has no code", addr.name, addr.addr, chainState.ID)
		}

		alloc := chainState.Allocs.Data.Accounts

		chainIntent := intent.Chains[i]

		// ownership slots
		var addrAsSlot common.Hash
		addrAsSlot.SetBytes(chainIntent.Roles.L1ProxyAdminOwner.Bytes())
		// slot 0
		ownerSlot := common.Hash{}
		checkStorageSlot(t, alloc, predeploys.ProxyAdminAddr, ownerSlot, addrAsSlot)

		if govEnabled {
			var defaultGovOwner common.Hash
			defaultGovOwner.SetBytes(common.HexToAddress("0xDeaDDEaDDeAdDeAdDEAdDEaddeAddEAdDEAdDEad").Bytes())
			checkStorageSlot(t, alloc, predeploys.GovernanceTokenAddr, common.Hash{31: 0x0a}, defaultGovOwner)
		} else {
			_, ok := alloc[predeploys.GovernanceTokenAddr]
			require.False(t, ok, "governance token should not be deployed by default")
		}

		genesis, rollup, err := inspect.GenesisAndRollup(st, chainState.ID)
		require.NoError(t, err)
		require.Equal(t, rollup.Genesis.SystemConfig.GasLimit, testCustomGasLimit, "rollup gasLimit")
		require.Equal(t, genesis.GasLimit, testCustomGasLimit, "genesis gasLimit")

		require.Equal(t, chainIntent.GasLimit, testCustomGasLimit, "chainIntent gasLimit")
		require.Equal(t, int(chainIntent.Eip1559Denominator), 50, "EIP1559Denominator should be set")
		require.Equal(t, int(chainIntent.Eip1559Elasticity), 6, "EIP1559Elasticity should be set")
	}
}

type bytesMarshaler interface {
	Bytes() []byte
}

func checkImmutable(t *testing.T, allocations types.GenesisAlloc, implementationAddress common.Address, thing bytesMarshaler) {
	account, ok := allocations[implementationAddress]
	require.True(t, ok, "%s not found in allocations", implementationAddress)
	require.NotEmpty(t, account.Code, "%s should have code", implementationAddress)
	require.True(
		t,
		bytes.Contains(account.Code, thing.Bytes()),
		"%s code should contain %s immutable", implementationAddress, hex.EncodeToString(thing.Bytes()),
	)
}

func checkStorageSlot(t *testing.T, allocs types.GenesisAlloc, address common.Address, slot common.Hash, expected common.Hash) {
	account, ok := allocs[address]
	require.True(t, ok, "account not found for address %s", address)
	value, ok := account.Storage[slot]
	if expected == (common.Hash{}) {
		require.False(t, ok, "slot %s for account %s should not be set", slot, address)
		return
	}
	require.True(t, ok, "slot %s not found for account %s", slot, address)
	require.Equal(t, expected, value, "slot %s for account %s should be %s", slot, address, expected)
}
