package integration_test

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"log/slog"
	"math/big"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/bootstrap"
	"github.com/ethereum/go-ethereum/params"

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

	op_e2e "github.com/ethereum-optimism/optimism/op-e2e"

	"github.com/holiman/uint256"

	"github.com/ethereum-optimism/optimism/op-chain-ops/addresses"
	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum-optimism/optimism/op-service/predeploys"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/require"
)

type deployerKey struct{}

func (d *deployerKey) HDPath() string {
	return "m/44'/60'/0'/0/0"
}

func (d *deployerKey) String() string {
	return "deployer-key"
}

func defaultPrivkey(t *testing.T) (string, *ecdsa.PrivateKey, *devkeys.MnemonicDevKeys) {
	pkHex := "ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
	pk, err := crypto.HexToECDSA(pkHex)
	require.NoError(t, err)

	dk, err := devkeys.NewMnemonicDevKeys(devkeys.TestMnemonic)
	require.NoError(t, err)

	return pkHex, pk, dk
}

// TestEndToEndBootstrapApply tests that a system can be fully bootstrapped and applied, both from
// local artifacts and the default tagged artifacts. The tagged artifacts test only runs on proposal
// or backports branches, since those are the only branches with an SLA to support tagged artifacts.
func TestEndToEndBootstrapApply(t *testing.T) {
	op_e2e.InitParallel(t)

	lgr := testlog.Logger(t, slog.LevelDebug)
	l1RPC, l1Client := devnet.DefaultAnvilRPC(t, lgr)
	pkHex, pk, dk := defaultPrivkey(t)
	l1ChainID := new(big.Int).SetUint64(devnet.DefaultChainID)
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
		require.NoError(t, err)

		var release string
		if !loc.IsTag() {
			release = "dev"
		}

		impls, err := bootstrap.Implementations(ctx, bootstrap.ImplementationsConfig{
			L1RPCUrl:                        l1RPC,
			PrivateKey:                      pkHex,
			ArtifactsLocator:                loc,
			L1ContractsRelease:              release,
			MIPSVersion:                     int(standard.MIPSVersion),
			WithdrawalDelaySeconds:          standard.WithdrawalDelaySeconds,
			MinProposalSizeBytes:            standard.MinProposalSizeBytes,
			ChallengePeriodSeconds:          standard.ChallengePeriodSeconds,
			ProofMaturityDelaySeconds:       standard.ProofMaturityDelaySeconds,
			DisputeGameFinalityDelaySeconds: standard.DisputeGameFinalityDelaySeconds,
			SuperchainConfigProxy:           bstrap.SuperchainConfigProxy,
			ProtocolVersionsProxy:           bstrap.ProtocolVersionsProxy,
			UpgradeController:               superchainPAO,
			SuperchainProxyAdmin:            bstrap.SuperchainProxyAdmin,
			CacheDir:                        testCacheDir,
			Logger:                          lgr,
		})
		require.NoError(t, err)

		intent, st := newIntent(t, l1ChainID, dk, l2ChainID, loc, loc)
		intent.SuperchainRoles = nil
		intent.OPCMAddress = &impls.Opcm

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
		op_e2e.InitParallel(t)
		testutils.RunOnBranch(t, regexp.MustCompile(`^(backports/op-deployer|proposal/op-contracts)/*`))
		apply(t, artifacts.DefaultL1ContractsLocator)
	})

	t.Run("local artifacts", func(t *testing.T) {
		op_e2e.InitParallel(t)
		loc, _ := testutil.LocalArtifacts(t)
		apply(t, loc)
	})
}

func TestEndToEndApply(t *testing.T) {
	op_e2e.InitParallel(t)

	lgr := testlog.Logger(t, slog.LevelDebug)
	l1RPC, l1Client := devnet.DefaultAnvilRPC(t, lgr)
	_, pk, dk := defaultPrivkey(t)
	l1ChainID := new(big.Int).SetUint64(devnet.DefaultChainID)
	l2ChainID1 := uint256.NewInt(1)
	l2ChainID2 := uint256.NewInt(2)
	loc, _ := testutil.LocalArtifacts(t)
	testCacheDir := testutils.IsolatedTestDirWithAutoCleanup(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Run("two chains one after another", func(t *testing.T) {
		intent, st := newIntent(t, l1ChainID, dk, l2ChainID1, loc, loc)
		cg := ethClientCodeGetter(ctx, l1Client)

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

		// create a new environment with wiped state to ensure we can continue using the
		// state from the previous deployment
		intent.Chains = append(intent.Chains, newChainIntent(t, dk, l1ChainID, l2ChainID2))

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

		validateSuperchainDeployment(t, st, cg, true)
		validateOPChainDeployment(t, cg, st, intent, false)
	})

	t.Run("with calldata broadcasts and prestate generation", func(t *testing.T) {
		intent, st := newIntent(t, l1ChainID, dk, l2ChainID1, loc, loc)
		mockPreStateBuilder := devnet.NewMockPreStateBuilder()

		require.NoError(t, deployer.ApplyPipeline(
			ctx,
			deployer.ApplyPipelineOpts{
				DeploymentTarget:   deployer.DeploymentTargetCalldata,
				L1RPCUrl:           l1RPC,
				DeployerPrivateKey: pk,
				Intent:             intent,
				State:              st,
				Logger:             lgr,
				StateWriter:        pipeline.NoopStateWriter(),
				CacheDir:           testCacheDir,
				PreStateBuilder:    mockPreStateBuilder,
			},
		))

		require.Greater(t, len(st.DeploymentCalldata), 0)
		require.Equal(t, 1, mockPreStateBuilder.Invocations())
		require.Equal(t, len(intent.Chains), mockPreStateBuilder.LastOptsCount())
		require.NotNil(t, st.PrestateManifest)
		for _, val := range *st.PrestateManifest {
			_, err := hexutil.Decode(val) // the not-empty val check is covered here as well
			require.NoError(t, err)
		}
	})
}

func TestGlobalOverrides(t *testing.T) {
	op_e2e.InitParallel(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	opts, intent, st := setupGenesisChain(t, devnet.DefaultChainID)
	expectedGasLimit := strings.ToLower("0x1C9C380")
	expectedBaseFeeVaultRecipient := common.HexToAddress("0x0000000000000000000000000000000000000001")
	expectedL1FeeVaultRecipient := common.HexToAddress("0x0000000000000000000000000000000000000002")
	expectedSequencerFeeVaultRecipient := common.HexToAddress("0x0000000000000000000000000000000000000003")
	expectedBaseFeeVaultMinimumWithdrawalAmount := strings.ToLower("0x1BC16D674EC80000")
	expectedBaseFeeVaultWithdrawalNetwork := genesis.FromUint8(0)
	expectedEnableGovernance := false
	expectedGasPriceOracleBaseFeeScalar := uint32(1300)
	expectedEIP1559Denominator := uint64(500)
	expectedUseFaultProofs := false
	intent.GlobalDeployOverrides = map[string]interface{}{
		"l2BlockTime":                         float64(3),
		"l2GenesisBlockGasLimit":              expectedGasLimit,
		"baseFeeVaultRecipient":               expectedBaseFeeVaultRecipient,
		"l1FeeVaultRecipient":                 expectedL1FeeVaultRecipient,
		"sequencerFeeVaultRecipient":          expectedSequencerFeeVaultRecipient,
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
	require.Equal(t, expectedGasLimit, strings.ToLower(cfg.L2InitializationConfig.L2GenesisBlockDeployConfig.L2GenesisBlockGasLimit.String()), "L2 Genesis Block Gas Limit should be 30_000_000")
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

func TestProofParamOverrides(t *testing.T) {
	op_e2e.InitParallel(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	opts, intent, st := setupGenesisChain(t, devnet.DefaultChainID)
	intent.GlobalDeployOverrides = map[string]any{
		"faultGameWithdrawalDelay":                standard.WithdrawalDelaySeconds + 1,
		"preimageOracleMinProposalSize":           standard.MinProposalSizeBytes + 1,
		"preimageOracleChallengePeriod":           standard.ChallengePeriodSeconds + 1,
		"proofMaturityDelaySeconds":               standard.ProofMaturityDelaySeconds + 1,
		"disputeGameFinalityDelaySeconds":         standard.DisputeGameFinalityDelaySeconds + 1,
		"mipsVersion":                             standard.MIPSVersion,     // Contract enforces a valid value be used
		"respectedGameType":                       standard.DisputeGameType, // This must be set to the permissioned game
		"faultGameAbsolutePrestate":               common.Hash{'A', 'B', 'S', 'O', 'L', 'U', 'T', 'E'},
		"faultGameMaxDepth":                       standard.DisputeMaxGameDepth + 1,
		"faultGameSplitDepth":                     standard.DisputeSplitDepth + 1,
		"faultGameClockExtension":                 standard.DisputeClockExtension + 1,
		"faultGameMaxClockDuration":               standard.DisputeMaxClockDuration + 1,
		"dangerouslyAllowCustomDisputeParameters": true,
	}

	require.NoError(t, deployer.ApplyPipeline(ctx, opts))

	allocs := st.L1StateDump.Data.Accounts
	chainState := st.Chains[0]

	uint64Caster := func(t *testing.T, val any) common.Hash {
		return common.BigToHash(new(big.Int).SetUint64(val.(uint64)))
	}

	tests := []struct {
		name    string
		caster  func(t *testing.T, val any) common.Hash
		address common.Address
	}{
		{
			"faultGameWithdrawalDelay",
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
			"proofMaturityDelaySeconds",
			uint64Caster,
			st.ImplementationsDeployment.OptimismPortalImpl,
		},
		{
			"disputeGameFinalityDelaySeconds",
			uint64Caster,
			st.ImplementationsDeployment.AnchorStateRegistryImpl,
		},
		{
			"faultGameAbsolutePrestate",
			func(t *testing.T, val any) common.Hash {
				return val.(common.Hash)
			},
			chainState.PermissionedDisputeGameImpl,
		},
		{
			"faultGameMaxDepth",
			uint64Caster,
			chainState.PermissionedDisputeGameImpl,
		},
		{
			"faultGameSplitDepth",
			uint64Caster,
			chainState.PermissionedDisputeGameImpl,
		},
		{
			"faultGameClockExtension",
			uint64Caster,
			chainState.PermissionedDisputeGameImpl,
		},
		{
			"faultGameMaxClockDuration",
			uint64Caster,
			chainState.PermissionedDisputeGameImpl,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checkImmutable(t, allocs, tt.address, tt.caster(t, intent.GlobalDeployOverrides[tt.name]))
		})
	}
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

			mockPreStateBuilder := devnet.NewMockPreStateBuilder()
			opts.PreStateBuilder = mockPreStateBuilder

			err := deployer.ApplyPipeline(ctx, opts)
			require.Error(t, err)
			require.ErrorContains(t, err, "failed to combine L2 init config")
			require.Equal(t, 0, mockPreStateBuilder.Invocations())
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
		"challengePeriodSeconds": 1,
	}
	intent.Chains[0].AdditionalDisputeGames = []state.AdditionalDisputeGame{
		{
			ChainProofParams: state.ChainProofParams{
				DisputeGameType:                         255,
				DisputeAbsolutePrestate:                 standard.DisputeAbsolutePrestate,
				DisputeMaxGameDepth:                     50,
				DisputeSplitDepth:                       14,
				DisputeClockExtension:                   0,
				DisputeMaxClockDuration:                 1200,
				DangerouslyAllowCustomDisputeParameters: true,
			},
			UseCustomOracle:              true,
			OracleMinProposalSize:        10000,
			OracleChallengePeriodSeconds: 120,
			MakeRespected:                true,
			VMType:                       state.VMTypeAlphabet,
		},
	}

	require.NoError(t, deployer.ApplyPipeline(ctx, opts))

	chainState := st.Chains[0]
	require.Equal(t, 1, len(chainState.AdditionalDisputeGames))

	gameInfo := chainState.AdditionalDisputeGames[0]
	require.NotEmpty(t, gameInfo.VMAddress)
	require.NotEmpty(t, gameInfo.GameAddress)
	require.NotEmpty(t, gameInfo.OracleAddress)
	require.NotEqual(t, st.ImplementationsDeployment.PreimageOracleImpl, gameInfo.OracleAddress)
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

	intent, st := newIntent(t, l1ChainIDBig, dk, l2ChainID1, loc, loc)

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

func addrFor(t *testing.T, dk *devkeys.MnemonicDevKeys, key devkeys.Key) common.Address {
	addr, err := dk.Address(key)
	require.NoError(t, err)
	return addr
}

func newIntent(
	t *testing.T,
	l1ChainID *big.Int,
	dk *devkeys.MnemonicDevKeys,
	l2ChainID *uint256.Int,
	l1Loc *artifacts.Locator,
	l2Loc *artifacts.Locator,
) (*state.Intent, *state.State) {
	intent := &state.Intent{
		ConfigType: state.IntentTypeCustom,
		L1ChainID:  l1ChainID.Uint64(),
		SuperchainRoles: &addresses.SuperchainRoles{
			SuperchainProxyAdminOwner: addrFor(t, dk, devkeys.L1ProxyAdminOwnerRole.Key(l1ChainID)),
			ProtocolVersionsOwner:     addrFor(t, dk, devkeys.SuperchainDeployerKey.Key(l1ChainID)),
			SuperchainGuardian:        addrFor(t, dk, devkeys.SuperchainConfigGuardianKey.Key(l1ChainID)),
		},
		FundDevAccounts:    false,
		L1ContractsLocator: l1Loc,
		L2ContractsLocator: l2Loc,
		Chains: []*state.ChainIntent{
			newChainIntent(t, dk, l1ChainID, l2ChainID),
		},
	}
	st := &state.State{
		Version: 1,
	}
	return intent, st
}

func newChainIntent(t *testing.T, dk *devkeys.MnemonicDevKeys, l1ChainID *big.Int, l2ChainID *uint256.Int) *state.ChainIntent {
	return &state.ChainIntent{
		ID:                         l2ChainID.Bytes32(),
		BaseFeeVaultRecipient:      addrFor(t, dk, devkeys.BaseFeeVaultRecipientRole.Key(l1ChainID)),
		L1FeeVaultRecipient:        addrFor(t, dk, devkeys.L1FeeVaultRecipientRole.Key(l1ChainID)),
		SequencerFeeVaultRecipient: addrFor(t, dk, devkeys.SequencerFeeVaultRecipientRole.Key(l1ChainID)),
		Eip1559DenominatorCanyon:   standard.Eip1559DenominatorCanyon,
		Eip1559Denominator:         standard.Eip1559Denominator,
		Eip1559Elasticity:          standard.Eip1559Elasticity,
		Roles: state.ChainRoles{
			L1ProxyAdminOwner: addrFor(t, dk, devkeys.L2ProxyAdminOwnerRole.Key(l1ChainID)),
			L2ProxyAdminOwner: addrFor(t, dk, devkeys.L2ProxyAdminOwnerRole.Key(l1ChainID)),
			SystemConfigOwner: addrFor(t, dk, devkeys.SystemConfigOwner.Key(l1ChainID)),
			UnsafeBlockSigner: addrFor(t, dk, devkeys.SequencerP2PRole.Key(l1ChainID)),
			Batcher:           addrFor(t, dk, devkeys.BatcherRole.Key(l1ChainID)),
			Proposer:          addrFor(t, dk, devkeys.ProposerRole.Key(l1ChainID)),
			Challenger:        addrFor(t, dk, devkeys.ChallengerRole.Key(l1ChainID)),
		},
	}
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

	if !intent.L1ContractsLocator.IsTag() {
		implAddrs = append(implAddrs, addrTuple{"EthLockboxImpl", st.ImplementationsDeployment.EthLockboxImpl})
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
		checkImmutableBehindProxy(t, alloc, predeploys.BaseFeeVaultAddr, chainIntent.BaseFeeVaultRecipient)
		checkImmutableBehindProxy(t, alloc, predeploys.L1FeeVaultAddr, chainIntent.L1FeeVaultRecipient)
		checkImmutableBehindProxy(t, alloc, predeploys.SequencerFeeVaultAddr, chainIntent.SequencerFeeVaultRecipient)
		checkImmutableBehindProxy(t, alloc, predeploys.OptimismMintableERC721FactoryAddr, common.BigToHash(new(big.Int).SetUint64(intent.L1ChainID)))

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

		require.Equal(t, int(chainIntent.Eip1559Denominator), 50, "EIP1559Denominator should be set")
		require.Equal(t, int(chainIntent.Eip1559Elasticity), 6, "EIP1559Elasticity should be set")
	}
}

func getEIP1967ImplementationAddress(t *testing.T, allocations types.GenesisAlloc, proxyAddress common.Address) common.Address {
	storage := allocations[proxyAddress].Storage
	storageValue := storage[genesis.ImplementationSlot]
	require.NotEmpty(t, storageValue, "Implementation address for %s should be set", proxyAddress)
	return common.HexToAddress(storageValue.Hex())
}

type bytesMarshaler interface {
	Bytes() []byte
}

func checkImmutableBehindProxy(t *testing.T, allocations types.GenesisAlloc, proxyContract common.Address, thing bytesMarshaler) {
	implementationAddress := getEIP1967ImplementationAddress(t, allocations, proxyContract)
	checkImmutable(t, allocations, implementationAddress, thing)
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
