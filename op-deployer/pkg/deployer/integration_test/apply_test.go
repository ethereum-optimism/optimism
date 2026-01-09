package integration_test

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/bootstrap"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/broadcaster"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/inspect"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/integration_test/shared"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/env"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/params"

	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/ethereum-optimism/optimism/op-service/testutils/devnet"

	altda "github.com/ethereum-optimism/optimism/op-alt-da"
	"github.com/ethereum-optimism/optimism/op-node/rollup"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/artifacts"

	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/pipeline"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/standard"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/testutil"
	opbindings "github.com/ethereum-optimism/optimism/op-e2e/bindings"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/upgrade/embedded"
	op_e2e "github.com/ethereum-optimism/optimism/op-e2e"

	"github.com/holiman/uint256"
	"github.com/lmittmann/w3"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/require"
)

const testCustomGasLimit = uint64(90_123_456)

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
			L1ProxyAdminOwner:               superchainPAO,
			SuperchainProxyAdmin:            bstrap.SuperchainProxyAdmin,
			CacheDir:                        testCacheDir,
			Logger:                          lgr,
			Challenger:                      common.Address{'C'},
			FaultGameMaxGameDepth:           standard.DisputeMaxGameDepth,
			FaultGameSplitDepth:             standard.DisputeSplitDepth,
			FaultGameClockExtension:         standard.DisputeClockExtension,
			FaultGameMaxClockDuration:       standard.DisputeMaxClockDuration,
		})
		require.NoError(t, err)

		intent, st := shared.NewIntent(t, l1ChainID, dk, l2ChainID, loc, loc, testCustomGasLimit)
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
		apply(t, artifacts.DefaultL1ContractsLocator)
	})

	t.Run("local artifacts", func(t *testing.T) {
		loc, _ := testutil.LocalArtifacts(t)
		apply(t, loc)
	})
}

// TestEndToEndBootstrapApplyWithUpgrade tests upgrading from a previous contracts release
// to embedded version of contracts by executing the following sequence:
//  1. create an anvil env that is a fork of op-sepolia
//  2. bootstrap.Implementations of the latest/embedded version of contracts, which will produce a new opcm
//  3. call opcm.upgradeSuperchainConfig on the opcm deployed in [2] (prerequisite for opcm.upgrade)
//  4. call opcm.upgrade on the opcm deployed in [2]
func TestEndToEndBootstrapApplyWithUpgrade(t *testing.T) {
	op_e2e.InitParallel(t)

	tests := []struct {
		name       string
		devFeature common.Hash
	}{
		{"default", common.Hash{}},
		{"opcm-v2", deployer.OPCMV2DevFlag},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			op_e2e.InitParallel(t)
			lgr := testlog.Logger(t, slog.LevelDebug)

			forkedL1, stopL1, err := devnet.NewForkedSepolia(lgr)
			require.NoError(t, err)
			pkHex, _, _ := shared.DefaultPrivkey(t)
			t.Cleanup(func() {
				require.NoError(t, stopL1())
			})
			loc, afactsFS := testutil.LocalArtifacts(t)
			testCacheDir := testutils.IsolatedTestDirWithAutoCleanup(t)

			superchain, err := standard.SuperchainFor(11155111)
			require.NoError(t, err)

			superchainProxyAdmin, err := standard.SuperchainProxyAdminAddrFor(11155111)
			require.NoError(t, err)

			superchainProxyAdminOwner, err := standard.L1ProxyAdminOwner(11155111)
			require.NoError(t, err)

			cfg := bootstrap.ImplementationsConfig{
				L1RPCUrl:                        forkedL1.RPCUrl(),
				PrivateKey:                      pkHex,
				ArtifactsLocator:                loc,
				MIPSVersion:                     int(standard.MIPSVersion),
				WithdrawalDelaySeconds:          standard.WithdrawalDelaySeconds,
				MinProposalSizeBytes:            standard.MinProposalSizeBytes,
				ChallengePeriodSeconds:          standard.ChallengePeriodSeconds,
				ProofMaturityDelaySeconds:       standard.ProofMaturityDelaySeconds,
				DisputeGameFinalityDelaySeconds: standard.DisputeGameFinalityDelaySeconds,
				DevFeatureBitmap:                tt.devFeature,
				SuperchainConfigProxy:           superchain.SuperchainConfigAddr,
				ProtocolVersionsProxy:           superchain.ProtocolVersionsAddr,
				L1ProxyAdminOwner:               superchainProxyAdminOwner,
				SuperchainProxyAdmin:            superchainProxyAdmin,
				CacheDir:                        testCacheDir,
				Logger:                          lgr,
				Challenger:                      common.Address{'C'},
				FaultGameMaxGameDepth:           standard.DisputeMaxGameDepth,
				FaultGameSplitDepth:             standard.DisputeSplitDepth,
				FaultGameClockExtension:         standard.DisputeClockExtension,
				FaultGameMaxClockDuration:       standard.DisputeMaxClockDuration,
			}
			if deployer.IsDevFeatureEnabled(tt.devFeature, deployer.OPCMV2DevFlag) {
				cfg.DevFeatureBitmap = deployer.OPCMV2DevFlag
			}

			runEndToEndBootstrapAndApplyUpgradeTest(t, afactsFS, cfg)
		})
	}
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

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Run("two chains one after another", func(t *testing.T) {
		intent, st := shared.NewIntent(t, l1ChainID, dk, l2ChainID1, loc, loc, testCustomGasLimit)
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
		intent.Chains = append(intent.Chains, shared.NewChainIntent(t, dk, l1ChainID, l2ChainID2, testCustomGasLimit))

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
		intent, st := shared.NewIntent(t, l1ChainID, dk, l2ChainID1, loc, loc, testCustomGasLimit)
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

	t.Run("with custom gas token", func(t *testing.T) {
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
			DeploymentTarget:   deployer.DeploymentTargetLive,
			L1RPCUrl:           l1RPC,
			DeployerPrivateKey: pk,
			Intent:             intent,
			State:              st,
			Logger:             lgr,
			StateWriter:        pipeline.NoopStateWriter(),
			CacheDir:           testCacheDir,
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

		// Enable OPCMV2 dev flag
		intent.GlobalDeployOverrides = map[string]any{
			"devFeatureBitmap": deployer.OPCMV2DevFlag,
		}

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

		// Verify that OPCMV2 was deployed in implementations
		require.NotEmpty(t, st.ImplementationsDeployment.OpcmV2Impl, "OPCMV2 implementation should be deployed")
		require.NotEmpty(t, st.ImplementationsDeployment.OpcmContainerImpl, "OPCM container implementation should be deployed")
		require.NotEmpty(t, st.ImplementationsDeployment.OpcmStandardValidatorImpl, "OPCM standard validator implementation should be deployed")

		// Verify that implementations are deployed on L1
		cg := ethClientCodeGetter(ctx, l1Client)

		opcmV2Code := cg(t, st.ImplementationsDeployment.OpcmV2Impl)
		require.NotEmpty(t, opcmV2Code, "OPCMV2 should have code deployed")

		// Verify that the dev feature bitmap is set to OPCMV2
		require.Equal(t, deployer.OPCMV2DevFlag, intent.GlobalDeployOverrides["devFeatureBitmap"])

		// Assert that the OPCM V1 addresses are zero
		require.Equal(t, common.Address{}, st.ImplementationsDeployment.OpcmImpl, "OPCM V1 implementation should be zero")
		require.Equal(t, common.Address{}, st.ImplementationsDeployment.OpcmContractsContainerImpl, "OPCM container implementation should be zero")
		require.Equal(t, common.Address{}, st.ImplementationsDeployment.OpcmGameTypeAdderImpl, "OPCM game type adder implementation should be zero")
		require.Equal(t, common.Address{}, st.ImplementationsDeployment.OpcmDeployerImpl, "OPCM deployer implementation should be zero")
		require.Equal(t, common.Address{}, st.ImplementationsDeployment.OpcmUpgraderImpl, "OPCM upgrader implementation should be zero")
		require.Equal(t, common.Address{}, st.ImplementationsDeployment.OpcmInteropMigratorImpl, "OPCM interop migrator implementation should be zero")
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
		"devFeatureBitmap":                        common.Hash{},
	}

	require.NoError(t, deployer.ApplyPipeline(ctx, opts))

	allocs := st.L1StateDump.Data.Accounts

	uint64Caster := func(t *testing.T, val any) common.Hash {
		return common.BigToHash(new(big.Int).SetUint64(val.(uint64)))
	}

	pdgImpl := st.ImplementationsDeployment.PermissionedDisputeGameV2Impl
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

func runEndToEndBootstrapAndApplyUpgradeTest(t *testing.T, afactsFS foundry.StatDirFs, implementationsConfig bootstrap.ImplementationsConfig) {
	lgr := implementationsConfig.Logger

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	superchainProxyAdminOwner := implementationsConfig.L1ProxyAdminOwner

	impls, err := bootstrap.Implementations(ctx, implementationsConfig)
	require.NoError(t, err)

	versionClient, err := ethclient.Dial(implementationsConfig.L1RPCUrl)
	require.NoError(t, err)
	defer versionClient.Close()

	shouldUpgradeSuperchainConfig, err := needsSuperchainConfigUpgrade(
		ctx,
		versionClient,
		implementationsConfig.SuperchainConfigProxy,
		impls.SuperchainConfigImpl,
	)
	require.NoError(t, err)

	// Now test the OPCM upgrade using the deployed impls.Opcm
	t.Run("opcm upgrade test", func(t *testing.T) {
		// Create script host for the upgrade
		rpcClient, err := rpc.Dial(implementationsConfig.L1RPCUrl)
		require.NoError(t, err)

		host, err := env.DefaultForkedScriptHost(
			ctx,
			broadcaster.NoopBroadcaster(),
			lgr,
			implementationsConfig.L1ProxyAdminOwner,
			afactsFS,
			rpcClient,
		)
		require.NoError(t, err)

		// Only run the superchain config upgrade if the live superchain config is behind the freshly deployed
		// implementation. Running the script when versions match will revert and panic the test harness.
		if shouldUpgradeSuperchainConfig {
			t.Run("upgrade superchain config", func(t *testing.T) {
				upgradeConfig := embedded.UpgradeSuperchainConfigInput{
					Prank:            superchainProxyAdminOwner,
					Opcm:             impls.Opcm,
					SuperchainConfig: implementationsConfig.SuperchainConfigProxy,
				}

				err = embedded.UpgradeSuperchainConfig(host, upgradeConfig)
				require.NoError(t, err, "Superchain config upgrade should succeed")
			})
		} else {
			t.Log("Skipping superchain config upgrade; onchain version is already up to date")
		}

		// Then run the OPCM upgrade
		t.Run("upgrade opcm", func(t *testing.T) {
			if deployer.IsDevFeatureEnabled(implementationsConfig.DevFeatureBitmap, deployer.OPCMV2DevFlag) {
				t.Skip("Skipping OPCM upgrade for OPCM V2")
				return
			}
			upgradeConfig := embedded.UpgradeOPChainInput{
				Prank: superchainProxyAdminOwner,
				Opcm:  impls.Opcm,
				ChainConfigs: []embedded.OPChainConfig{
					{
						SystemConfigProxy:  deployer.DefaultSystemConfigProxySepolia,
						CannonPrestate:     common.Hash{'C', 'A', 'N', 'N', 'O', 'N'},
						CannonKonaPrestate: common.Hash{'K', 'O', 'N', 'A'},
					},
				},
			}
			// Test the upgrade
			upgradeConfigBytes, err := json.Marshal(upgradeConfig)
			require.NoError(t, err, "UpgradeOPChainInput should marshal to JSON")
			err = embedded.DefaultUpgrader.Upgrade(host, upgradeConfigBytes)
			require.NoError(t, err, "OPCM upgrade should succeed")
		})
		t.Run("upgrade opcm v2", func(t *testing.T) {
			if !deployer.IsDevFeatureEnabled(implementationsConfig.DevFeatureBitmap, deployer.OPCMV2DevFlag) {
				t.Skip("Skipping OPCM V2 upgrade for non-OPCM V2 dev feature")
				return
			}
			require.NotEqual(t, common.Address{}, impls.OpcmV2, "OpcmV2 address should not be zero")
			t.Logf("Using OpcmV2 at address: %s", impls.OpcmV2.Hex())
			t.Logf("Using OpcmUtils at address: %s", impls.OpcmUtils.Hex())
			t.Logf("Using OpcmContainer at address: %s", impls.OpcmContainer.Hex())

			// Verify OPCM V2 has code deployed
			opcmCode, err := versionClient.CodeAt(ctx, impls.OpcmV2, nil)
			require.NoError(t, err)
			require.NotEmpty(t, opcmCode, "OPCM V2 should have code deployed")
			t.Logf("OPCM V2 code size: %d bytes", len(opcmCode))

			// Verify OpcmUtils has code deployed
			utilsCode, err := versionClient.CodeAt(ctx, impls.OpcmUtils, nil)
			require.NoError(t, err)
			require.NotEmpty(t, utilsCode, "OpcmUtils should have code deployed")
			t.Logf("OpcmUtils code size: %d bytes", len(utilsCode))

			// Verify OpcmContainer has code deployed
			containerCode, err := versionClient.CodeAt(ctx, impls.OpcmContainer, nil)
			require.NoError(t, err)
			require.NotEmpty(t, containerCode, "OpcmContainer should have code deployed")
			t.Logf("OpcmContainer code size: %d bytes", len(containerCode))

			// First, upgrade the superchain with V2
			t.Run("upgrade superchain v2", func(t *testing.T) {
				superchainUpgradeConfig := embedded.UpgradeSuperchainConfigInput{
					Prank:             superchainProxyAdminOwner,
					Opcm:              impls.OpcmV2,
					SuperchainConfig:  implementationsConfig.SuperchainConfigProxy,
					ExtraInstructions: []embedded.ExtraInstruction{},
				}
				err := embedded.UpgradeSuperchainConfig(host, superchainUpgradeConfig)
				if err != nil {
					t.Logf("Superchain upgrade may have failed (could already be upgraded): %v", err)
				} else {
					t.Log("Superchain V2 upgrade succeeded")
				}
			})

			// Then test upgrade on the V2-deployed chain
			t.Run("upgrade chain v2", func(t *testing.T) {
				// ABI-encode game args for FaultDisputeGameConfig{absolutePrestate}
				bytes32Type := deployer.Bytes32Type
				addressType := deployer.AddressType

				// FaultDisputeGameConfig just needs absolutePrestate (bytes32)
				testPrestate := common.Hash{'P', 'R', 'E', 'S', 'T', 'A', 'T', 'E'}
				cannonArgs, err := abi.Arguments{{Type: bytes32Type}}.Pack(testPrestate)
				require.NoError(t, err)

				// PermissionedDisputeGameConfig needs absolutePrestate, proposer, challenger
				testProposer := common.Address{'P'}
				testChallenger := common.Address{'C'}
				permissionedArgs, err := abi.Arguments{
					{Type: bytes32Type},
					{Type: addressType},
					{Type: addressType},
				}.Pack(testPrestate, testProposer, testChallenger)
				require.NoError(t, err)

				upgradeConfig := embedded.UpgradeOPChainInput{
					Prank: superchainProxyAdminOwner,
					Opcm:  impls.OpcmV2,
					UpgradeInputV2: &embedded.UpgradeInputV2{
						SystemConfig: deployer.DefaultSystemConfigProxySepolia,
						DisputeGameConfigs: []embedded.DisputeGameConfig{
							{
								Enabled:  true,
								InitBond: big.NewInt(1000000000000000000),
								GameType: embedded.GameTypeCannon,
								GameArgs: cannonArgs,
							},
							{
								Enabled:  true,
								InitBond: big.NewInt(1000000000000000000),
								GameType: embedded.GameTypePermissionedCannon,
								GameArgs: permissionedArgs,
							},
							{
								Enabled:  false,
								InitBond: big.NewInt(0),
								GameType: embedded.GameTypeCannonKona,
								GameArgs: []byte{}, // Disabled games don't need args
							},
						},
						ExtraInstructions: []embedded.ExtraInstruction{
							{
								Key:  "PermittedProxyDeployment",
								Data: []byte("DelayedWETH"),
							},
							// TODO(#18502): Remove the extra instruction for custom gas token after U18 ships.
							{
								Key:  "overrides.cfg.useCustomGasToken",
								Data: make([]byte, 32),
							},
						},
					},
				}

				upgradeConfigBytes, err := json.Marshal(upgradeConfig)
				require.NoError(t, err, "UpgradeOPChainV2Input should marshal to JSON")

				// Verify input encoding
				encodedData, err := upgradeConfig.EncodedUpgradeInputV2()
				require.NoError(t, err, "Should encode UpgradeInputV2")
				require.NotEmpty(t, encodedData, "Encoded data should not be empty")

				// Build expected hex encoding
				// Structure breakdown:
				// - Tuple offset (0x20)
				// - SystemConfig address (0x034edd2a225f7f429a63e0f1d2084b9e0a93b538)
				// - DisputeGameConfigs array offset (0x60) and ExtraInstructions array offset (0x340)
				// - DisputeGameConfigs[]: 3 configs
				//   [0] Cannon: enabled=true, initBond=1e18, gameType=0, gameArgs="PRESTATE"
				//   [1] PermissionedCannon: enabled=true, initBond=1e18, gameType=1, gameArgs="PRESTATE"+proposer+challenger
				//   [2] CannonKona: enabled=false, initBond=0, gameType=0, gameArgs=empty
				// - ExtraInstructions[]: 2 instructions
				//   [0] key="PermittedProxyDeployment", data="DelayedWETH"
				//   [1] key="overrides.cfg.useCustomGasToken", data=32 zero bytes
				expected := "0000000000000000000000000000000000000000000000000000000000000020" + // offset to tuple
					"000000000000000000000000034edd2a225f7f429a63e0f1d2084b9e0a93b538" + // systemConfig address
					"0000000000000000000000000000000000000000000000000000000000000060" + // offset to disputeGameConfigs
					"0000000000000000000000000000000000000000000000000000000000000340" + // offset to extraInstructions
					"0000000000000000000000000000000000000000000000000000000000000003" + // disputeGameConfigs.length (3)
					"0000000000000000000000000000000000000000000000000000000000000060" + // offset to disputeGameConfigs[0]
					"0000000000000000000000000000000000000000000000000000000000000120" + // offset to disputeGameConfigs[1]
					"0000000000000000000000000000000000000000000000000000000000000220" + // offset to disputeGameConfigs[2]
					// DisputeGameConfigs[0] - Cannon
					"0000000000000000000000000000000000000000000000000000000000000001" + // enabled=true
					"0000000000000000000000000000000000000000000000000de0b6b3a7640000" + // initBond=1e18
					"0000000000000000000000000000000000000000000000000000000000000000" + // gameType=0 (Cannon)
					"0000000000000000000000000000000000000000000000000000000000000080" + // offset to gameArgs
					"0000000000000000000000000000000000000000000000000000000000000020" + // gameArgs.length (32 bytes)
					"5052455354415445000000000000000000000000000000000000000000000000" + // gameArgs data "PRESTATE"
					// DisputeGameConfigs[1] - PermissionedCannon
					"0000000000000000000000000000000000000000000000000000000000000001" + // enabled=true
					"0000000000000000000000000000000000000000000000000de0b6b3a7640000" + // initBond=1e18
					"0000000000000000000000000000000000000000000000000000000000000001" + // gameType=1 (PermissionedCannon)
					"0000000000000000000000000000000000000000000000000000000000000080" + // offset to gameArgs
					"0000000000000000000000000000000000000000000000000000000000000060" + // gameArgs.length (96 bytes)
					"5052455354415445000000000000000000000000000000000000000000000000" + // gameArgs data "PRESTATE"
					"0000000000000000000000005000000000000000000000000000000000000000" + // proposer address
					"0000000000000000000000004300000000000000000000000000000000000000" + // challenger address
					// DisputeGameConfigs[2] - CannonKona (disabled)
					"0000000000000000000000000000000000000000000000000000000000000000" + // enabled=false
					"0000000000000000000000000000000000000000000000000000000000000000" + // initBond=0
					"0000000000000000000000000000000000000000000000000000000000000008" + // gameType=8 (CannonKona)
					"0000000000000000000000000000000000000000000000000000000000000080" + // offset to gameArgs
					"0000000000000000000000000000000000000000000000000000000000000000" + // gameArgs.length (0)
					// ExtraInstructions array
					"0000000000000000000000000000000000000000000000000000000000000002" + // extraInstructions.length (2)
					"0000000000000000000000000000000000000000000000000000000000000040" + // offset to extraInstructions[0]
					"0000000000000000000000000000000000000000000000000000000000000100" + // offset to extraInstructions[1]
					// ExtraInstructions[0] - PermittedProxyDeployment
					"0000000000000000000000000000000000000000000000000000000000000040" + // offset to key
					"0000000000000000000000000000000000000000000000000000000000000080" + // offset to data
					"0000000000000000000000000000000000000000000000000000000000000018" + // key.length (24 bytes)
					"5065726d697474656450726f78794465706c6f796d656e74000000000000000" + // "PermittedProxyDeployment"
					"0" + // padding
					"000000000000000000000000000000000000000000000000000000000000000b" + // data.length (11 bytes)
					"44656c617965645745544800000000000000000000000000000000000000000" + // "DelayedWETH"
					"0" + // padding
					// ExtraInstructions[1] - useCustomGasToken override
					"0000000000000000000000000000000000000000000000000000000000000040" + // offset to key
					"0000000000000000000000000000000000000000000000000000000000000080" + // offset to data
					"000000000000000000000000000000000000000000000000000000000000001f" + // key.length (31 bytes)
					"6f76657272696465732e6366672e757365437573746f6d476173546f6b656e00" + // "overrides.cfg.useCustomGasToken"
					"0000000000000000000000000000000000000000000000000000000000000020" + // data.length (32 bytes)
					"0000000000000000000000000000000000000000000000000000000000000000" // data (32 zero bytes)

				require.Equal(t, expected, hex.EncodeToString(encodedData), "Encoded calldata should match expected structure")

				err = embedded.DefaultUpgrader.Upgrade(host, upgradeConfigBytes)
				require.NoError(t, err, "OPCM V2 chain upgrade should succeed")
			})
		})
	})
}

func needsSuperchainConfigUpgrade(
	ctx context.Context,
	client *ethclient.Client,
	currentProxy, targetImpl common.Address,
) (bool, error) {
	currentVersion, err := superchainConfigVersion(ctx, client, currentProxy)
	if err != nil {
		return false, fmt.Errorf("failed to fetch proxy superchain config version: %w", err)
	}

	targetVersion, err := superchainConfigVersion(ctx, client, targetImpl)
	if err != nil {
		return false, fmt.Errorf("failed to fetch implementation superchain config version: %w", err)
	}

	return currentVersion.LessThan(targetVersion), nil
}

func superchainConfigVersion(
	ctx context.Context,
	client *ethclient.Client,
	addr common.Address,
) (*semver.Version, error) {
	contract, err := opbindings.NewSuperchainConfig(addr, client)
	if err != nil {
		return nil, fmt.Errorf("failed to bind superchain config at %s: %w", addr.Hex(), err)
	}
	versionStr, err := contract.Version(&bind.CallOpts{Context: ctx})
	if err != nil {
		return nil, fmt.Errorf("failed to read version from %s: %w", addr.Hex(), err)
	}
	version, err := semver.NewVersion(versionStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse version %q from %s: %w", versionStr, addr.Hex(), err)
	}
	return version, nil
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

		genesis, rollup, err := inspect.GenesisAndRollup(st, chainState.ID)
		require.NoError(t, err)
		require.Equal(t, rollup.Genesis.SystemConfig.GasLimit, testCustomGasLimit, "rollup gasLimit")
		require.Equal(t, genesis.GasLimit, testCustomGasLimit, "genesis gasLimit")

		require.Equal(t, chainIntent.GasLimit, testCustomGasLimit, "chainIntent gasLimit")
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
