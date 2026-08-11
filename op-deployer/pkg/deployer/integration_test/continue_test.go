package integration_test

import (
	"context"
	"log/slog"
	"math/big"
	"path/filepath"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-core/devfeatures"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/bootstrap"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/integration_test/shared"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/pipeline"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/standard"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/testutil"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/upgrade/embedded"
	op_e2e "github.com/ethereum-optimism/optimism/op-e2e"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/ethereum-optimism/optimism/op-service/testutils/devnet"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

type continuationEnv struct {
	ctx                    context.Context
	lgr                    log.Logger
	logs                   *testlog.CapturingHandler
	l1RPC                  string
	l1Client               *ethclient.Client
	privateKey             string
	deployer               common.Address
	cacheDir               string
	workdir                string
	intent                 *state.Intent
	prepared               *state.State
	preparedChain          *state.ChainState
	preparedChains         []*state.ChainState
	preparedSnapshotChain  *state.PreparedChainState
	preparedSnapshotChains []*state.PreparedChainState
	opcm                   common.Address
	standardValidator      common.Address
}

func TestEndToEndContinuePreparedChain(t *testing.T) {
	op_e2e.InitParallel(t)

	t.Run("permissionless", func(t *testing.T) {
		testContinuePermissionless(t)
	})
	t.Run("permissionless output-root bootstrap", func(t *testing.T) {
		testContinuePermissionlessOutputRoot(t)
	})
	t.Run("permissionless with custom roles", func(t *testing.T) {
		testContinuePermissionlessWithCustomRoles(t)
	})
	t.Run("prestate override drift is rejected before broadcast", func(t *testing.T) {
		testContinueRejectsPrestateOverrideDrift(t)
	})
	t.Run("permissioned without committed prestate", func(t *testing.T) {
		testContinuePermissioned(t)
	})
	t.Run("permissioned with super-root OPCM", func(t *testing.T) {
		testContinuePermissionedSuperRoot(t)
	})
	t.Run("live validation failure persists checkpoint", func(t *testing.T) {
		testContinueLiveValidationFailure(t)
	})
	t.Run("post-checkpoint reorg redeploys", func(t *testing.T) {
		testContinuePostCheckpointReorg(t)
	})
	t.Run("partial deployment is rejected", func(t *testing.T) {
		testContinuePartialDeployment(t)
	})
	t.Run("multi-chain global preflight and mixed modes", func(t *testing.T) {
		testContinueMultiChainGlobalPreflight(t)
	})
	t.Run("later live validation failure preserves checkpoints", func(t *testing.T) {
		testContinueMultiChainLiveValidationFailure(t)
	})
	t.Run("live validation gates the next send", func(t *testing.T) {
		testContinueMultiChainSequentialValidation(t)
	})
	t.Run("mixed permissionless families are rejected", func(t *testing.T) {
		testContinueRejectsMixedPermissionlessFamilies(t)
	})
}

func testContinuePermissionless(t *testing.T) {
	t.Helper()
	env := newContinuationEnv(t, embedded.GameTypeSuperCannonKona)
	env.preparedChain.Prestate = common.HexToHash("0x1234")
	env.preparedChain.StartingAnchorRoot = &state.StartingAnchorProposal{
		Root:             common.HexToHash("0x5678"),
		L2SequenceNumber: 7,
	}
	originalAnchor := env.preparedSnapshotChain.StartBlock.Hash
	originalContracts := env.preparedSnapshotChain.OpChainContracts
	require.NoError(t, pipeline.WriteState(env.workdir, env.prepared))
	nonceBefore := pendingNonce(t, env)

	env.preparedSnapshotChain.StartBlock.Hash = common.HexToHash("0xdead")
	require.NoError(t, pipeline.WriteState(env.workdir, env.prepared))
	err := deployer.Continue(env.ctx, env.config())
	require.ErrorContains(t, err, "pinned anchor block")
	require.Equal(t, nonceBefore, pendingNonce(t, env))
	env.preparedSnapshotChain.StartBlock.Hash = originalAnchor

	validatorCode, err := env.l1Client.CodeAt(env.ctx, env.standardValidator, nil)
	require.NoError(t, err)
	setAnvilCode(t, env.l1Client, env.standardValidator, validatorResultCode("TEST-FAIL"))
	require.NoError(t, pipeline.WriteState(env.workdir, env.prepared))
	err = deployer.Continue(env.ctx, env.config())
	require.ErrorContains(t, err, "standard validator reported errors: TEST-FAIL")
	require.Equal(t, nonceBefore, pendingNonce(t, env))
	setAnvilCode(t, env.l1Client, env.standardValidator, validatorCode)

	// The game mode read is the first call made against the pinned OPCM, so an unusable
	// OPCM stops the run before the fork simulation.
	opcmCode, err := env.l1Client.CodeAt(env.ctx, env.opcm, nil)
	require.NoError(t, err)
	setAnvilCode(t, env.l1Client, env.opcm, []byte{byte(vm.PUSH0), byte(vm.PUSH0), byte(vm.REVERT)})
	require.NoError(t, pipeline.WriteState(env.workdir, env.prepared))
	err = deployer.Continue(env.ctx, env.config())
	require.ErrorContains(t, err, "failed to read the OPCM game mode")
	require.ErrorContains(t, err, env.opcm.Hex())
	require.Equal(t, nonceBefore, pendingNonce(t, env))
	setAnvilCode(t, env.l1Client, env.opcm, opcmCode)

	env.preparedSnapshotChain.SystemConfigProxy = common.Address{0xff}
	require.NoError(t, pipeline.WriteState(env.workdir, env.prepared))
	err = deployer.Continue(env.ctx, env.config())
	require.ErrorContains(t, err, "simulated contract addresses differ")
	require.Equal(t, nonceBefore, pendingNonce(t, env))

	env.preparedSnapshotChain.OpChainContracts = originalContracts
	require.NoError(t, pipeline.WriteState(env.workdir, env.prepared))
	require.NoError(t, deployer.Continue(env.ctx, env.config()))
	continued := assertContinuationCompleted(t, env, nonceBefore)
	continuedChain, err := continued.Chain(env.intent.Chains[0].ID)
	require.NoError(t, err)
	recordedContracts := continuedChain.OpChainContracts

	nonceAfter := pendingNonce(t, env)
	require.NoError(t, deployer.Continue(env.ctx, env.config()))
	require.Equal(t, nonceAfter, pendingNonce(t, env))

	require.NoError(t, pipeline.WriteState(env.workdir, env.prepared))
	require.NoError(t, deployer.Continue(env.ctx, env.config()))
	require.Equal(t, nonceAfter, pendingNonce(t, env))
	reconciled, err := pipeline.ReadState(env.workdir)
	require.NoError(t, err)
	reconciledChain, err := reconciled.Chain(env.intent.Chains[0].ID)
	require.NoError(t, err)
	require.Equal(t, recordedContracts, reconciledChain.OpChainContracts)
	require.Equal(t, originalContracts, env.preparedSnapshotChain.OpChainContracts)
	require.NotNil(t, reconciledChain.Continuation)
	require.Nil(t, reconciled.AppliedIntent)
}

func testContinuePermissionlessOutputRoot(t *testing.T) {
	t.Helper()
	env := newContinuationEnvWithIntentMutator(
		t,
		[]embedded.GameType{embedded.GameTypeCannonKona},
		devfeatures.OutputRootGamesFlag,
		func(intent *state.Intent) {
			intent.Chains[0].DeployOverrides = map[string]any{
				state.FaultGameAbsolutePrestateOverrideKey: standard.DisputeAbsolutePrestate,
			}
		},
	)
	require.NotNil(t, env.preparedChain.StartingAnchorRoot)
	require.NotZero(t, env.preparedChain.StartingAnchorRoot.Root)
	require.NotEqual(t, opcm.DefaultStartingAnchorRoot.Root, env.preparedChain.StartingAnchorRoot.Root)
	require.Zero(t, env.preparedChain.StartingAnchorRoot.L2SequenceNumber)

	require.NoError(t, deployer.Prestate(env.ctx, deployer.PrestateConfig{
		Workdir: env.workdir,
		Logger:  env.lgr,
	}))
	nonceBefore := pendingNonce(t, env)
	require.NoError(t, deployer.Continue(env.ctx, env.config()))
	assertContinuationCompleted(t, env, nonceBefore)
}

func testContinuePermissionlessWithCustomRoles(t *testing.T) {
	t.Helper()
	customProxyAdminOwner := common.Address{0xa1}
	customChallenger := common.Address{0xa2}
	env := newContinuationEnvWithIntentMutator(
		t,
		[]embedded.GameType{embedded.GameTypeSuperCannonKona},
		common.Hash{},
		func(intent *state.Intent) {
			roles := &intent.Chains[0].Roles
			require.NotEqual(t, roles.L1ProxyAdminOwner, customProxyAdminOwner)
			require.NotEqual(t, roles.Challenger, customChallenger)
			roles.L1ProxyAdminOwner = customProxyAdminOwner
			roles.Challenger = customChallenger
		},
	)
	setPermissionlessContinuationInputs(env.preparedChain, 7)
	require.NoError(t, pipeline.WriteState(env.workdir, env.prepared))
	nonceBefore := pendingNonce(t, env)

	require.NoError(t, deployer.Continue(env.ctx, env.config()))
	assertContinuationCompleted(t, env, nonceBefore)
}

func testContinueRejectsPrestateOverrideDrift(t *testing.T) {
	t.Helper()
	env := newContinuationEnv(t, embedded.GameTypeSuperCannonKona)
	prestateA := common.HexToHash("0x1234")
	env.intent.Chains[0].DeployOverrides[state.FaultGameAbsolutePrestateOverrideKey] = prestateA
	require.NoError(t, env.intent.WriteToFile(filepath.Join(env.workdir, "intent.toml")))
	require.NoError(t, deployer.Prestate(env.ctx, deployer.PrestateConfig{
		Workdir: env.workdir,
		Logger:  env.lgr,
	}))
	committed, err := pipeline.ReadState(env.workdir)
	require.NoError(t, err)
	committedChain, err := committed.Chain(env.intent.Chains[0].ID)
	require.NoError(t, err)
	require.Equal(t, prestateA, committedChain.Prestate)

	env.intent.Chains[0].DeployOverrides[state.FaultGameAbsolutePrestateOverrideKey] = common.HexToHash("0x5678")
	require.NoError(t, env.intent.WriteToFile(filepath.Join(env.workdir, "intent.toml")))
	nonceBefore := pendingNonce(t, env)
	err = deployer.Continue(env.ctx, env.config())
	require.ErrorContains(t, err, "override differs from the committed prestate")
	require.ErrorContains(t, err, "Rerun op-deployer prestate")
	require.Equal(t, nonceBefore, pendingNonce(t, env))
}

func testContinuePermissioned(t *testing.T) {
	t.Helper()
	env := newContinuationEnv(t, embedded.GameTypeSuperPermissioned)
	require.Zero(t, env.preparedChain.Prestate)
	elapsedGenesisTime := hexutil.Uint64(1)
	env.preparedSnapshotChain.GenesisTime = &elapsedGenesisTime
	require.NoError(t, pipeline.WriteState(env.workdir, env.prepared))
	nonceBefore := pendingNonce(t, env)

	require.NoError(t, deployer.Continue(env.ctx, env.config()))
	continued := assertContinuationCompleted(t, env, nonceBefore)
	env.logs.RequireMessageContained(
		t,
		"committed genesis time has elapsed",
		testlog.NewLevelFilter(slog.LevelWarn),
		testlog.NewAttributesFilter("chainID", env.intent.Chains[0].ID.Hex()),
	)

	continuedChain, err := continued.Chain(env.intent.Chains[0].ID)
	require.NoError(t, err)
	setAnvilCode(t, env.l1Client, continuedChain.SystemConfigProxy, nil)
	nonceAfter := pendingNonce(t, env)
	err = deployer.Continue(env.ctx, env.config())
	require.ErrorContains(t, err, "partial deployment at predicted addresses")
	require.ErrorContains(t, err, "SystemConfigProxy")
	require.Equal(t, nonceAfter, pendingNonce(t, env))
}

func testContinuePermissionedSuperRoot(t *testing.T) {
	t.Helper()
	env := newContinuationEnvForGameTypesAndFeatures(
		t,
		[]embedded.GameType{embedded.GameTypeSuperPermissioned},
		devfeatures.SuperRootGamesMigrationFlag,
	)
	require.Zero(t, env.preparedChain.Prestate)
	require.NoError(t, pipeline.WriteState(env.workdir, env.prepared))
	nonceBefore := pendingNonce(t, env)

	require.NoError(t, deployer.Continue(env.ctx, env.config()))
	assertContinuationCompleted(t, env, nonceBefore)
}

func testContinueLiveValidationFailure(t *testing.T) {
	t.Helper()
	env := newContinuationEnv(t, embedded.GameTypeSuperCannonKona)
	env.preparedChain.Prestate = common.HexToHash("0x1234")
	env.preparedChain.StartingAnchorRoot = &state.StartingAnchorProposal{
		Root:             common.HexToHash("0x5678"),
		L2SequenceNumber: 7,
	}
	require.NoError(t, pipeline.WriteState(env.workdir, env.prepared))
	nonceBefore := pendingNonce(t, env)
	latest, err := env.l1Client.HeaderByNumber(env.ctx, nil)
	require.NoError(t, err)
	liveValidationBlock := new(big.Int).Add(latest.Number, big.NewInt(1))
	setAnvilCode(t, env.l1Client, env.standardValidator, conditionalValidatorCode(liveValidationBlock))

	err = deployer.Continue(env.ctx, env.config())
	require.ErrorContains(t, err, "live deployment validation failed")
	require.ErrorContains(t, err, "TEST-FAIL")
	require.Equal(t, nonceBefore+1, pendingNonce(t, env))

	continued, err := pipeline.ReadState(env.workdir)
	require.NoError(t, err)
	require.True(t, continued.IsChainDeployed(env.intent.Chains[0].ID))
	require.Nil(t, continued.AppliedIntent)
	continuedChain, err := continued.Chain(env.intent.Chains[0].ID)
	require.NoError(t, err)
	require.NotNil(t, continuedChain.Continuation)
	continuedSnapshot, err := continued.PreparedDeployment.Chain(env.intent.Chains[0].ID)
	require.NoError(t, err)
	require.Equal(t, env.preparedSnapshotChain.OpChainContracts, continuedSnapshot.OpChainContracts)

	setAnvilCode(t, env.l1Client, env.standardValidator, validatorResultCode(validValidatorResult))
	nonceAfterFailure := pendingNonce(t, env)
	require.NoError(t, deployer.Continue(env.ctx, env.config()))
	require.Equal(t, nonceAfterFailure, pendingNonce(t, env))
	retried, err := pipeline.ReadState(env.workdir)
	require.NoError(t, err)
	retriedChain, err := retried.Chain(env.intent.Chains[0].ID)
	require.NoError(t, err)
	require.NotNil(t, retriedChain.Continuation)
	require.Nil(t, retried.AppliedIntent)
}

func testContinuePostCheckpointReorg(t *testing.T) {
	t.Helper()
	env := newContinuationEnv(t, embedded.GameTypeSuperPermissioned)
	nonceBefore := pendingNonce(t, env)
	var snapshotID string
	require.NoError(t, env.l1Client.Client().Call(&snapshotID, "evm_snapshot"))
	require.NotEmpty(t, snapshotID)

	require.NoError(t, deployer.Continue(env.ctx, env.config()))
	assertContinuationCompleted(t, env, nonceBefore)

	var reverted bool
	require.NoError(t, env.l1Client.Client().Call(&reverted, "evm_revert", snapshotID))
	require.True(t, reverted)
	require.Equal(t, nonceBefore, pendingNonce(t, env))
	code, err := env.l1Client.CodeAt(env.ctx, env.preparedSnapshotChain.SystemConfigProxy, nil)
	require.NoError(t, err)
	require.Empty(t, code)

	require.NoError(t, deployer.Continue(env.ctx, env.config()))
	assertContinuationCompleted(t, env, nonceBefore)
	code, err = env.l1Client.CodeAt(env.ctx, env.preparedSnapshotChain.SystemConfigProxy, nil)
	require.NoError(t, err)
	require.NotEmpty(t, code)

	nonceAfterRecovery := pendingNonce(t, env)
	require.NoError(t, deployer.Continue(env.ctx, env.config()))
	require.Equal(t, nonceAfterRecovery, pendingNonce(t, env))
}

func testContinuePartialDeployment(t *testing.T) {
	t.Helper()
	env := newContinuationEnv(t, embedded.GameTypeSuperCannonKona)
	env.preparedChain.Prestate = common.HexToHash("0x1234")
	env.preparedChain.StartingAnchorRoot = &state.StartingAnchorProposal{
		Root:             common.HexToHash("0x5678"),
		L2SequenceNumber: 7,
	}
	setAnvilCode(t, env.l1Client, env.preparedChain.SystemConfigProxy, []byte{byte(vm.STOP)})
	require.NoError(t, pipeline.WriteState(env.workdir, env.prepared))
	nonceBefore := pendingNonce(t, env)

	err := deployer.Continue(env.ctx, env.config())
	require.ErrorContains(t, err, "partial deployment at predicted addresses")
	require.ErrorContains(t, err, "SystemConfigProxy")
	require.Equal(t, nonceBefore, pendingNonce(t, env))
}

func testContinueMultiChainGlobalPreflight(t *testing.T) {
	t.Helper()
	env := newContinuationEnvForGameTypes(t, []embedded.GameType{
		embedded.GameTypeSuperPermissioned,
		embedded.GameTypeSuperCannonKona,
	})
	permissionless := env.preparedChains[1]
	setPermissionlessContinuationInputs(permissionless, 2)
	preparedPermissionless := env.preparedSnapshotChains[1]
	originalSystemConfig := preparedPermissionless.SystemConfigProxy
	preparedPermissionless.SystemConfigProxy = common.Address{0xff}
	require.NoError(t, pipeline.WriteState(env.workdir, env.prepared))
	nonceBefore := pendingNonce(t, env)

	err := deployer.Continue(env.ctx, env.config())
	require.ErrorContains(t, err, "simulated contract addresses differ")
	require.Equal(t, nonceBefore, pendingNonce(t, env))
	failed, err := pipeline.ReadState(env.workdir)
	require.NoError(t, err)
	for _, chain := range env.intent.Chains {
		require.False(t, failed.IsChainDeployed(chain.ID))
	}

	preparedPermissionless.SystemConfigProxy = originalSystemConfig
	require.NoError(t, pipeline.WriteState(env.workdir, env.prepared))
	require.NoError(t, deployer.Continue(env.ctx, env.config()))
	require.Equal(t, nonceBefore+uint64(len(env.intent.Chains)), pendingNonce(t, env))
	continued, err := pipeline.ReadState(env.workdir)
	require.NoError(t, err)
	require.Nil(t, continued.AppliedIntent)
	for i, chain := range env.intent.Chains {
		require.True(t, continued.IsChainDeployed(chain.ID))
		continuedChain, chainErr := continued.Chain(chain.ID)
		require.NoError(t, chainErr)
		require.NotNil(t, continuedChain.Continuation)
		if i == 0 {
			require.Zero(t, continuedChain.Prestate)
		} else {
			require.NotZero(t, continuedChain.Prestate)
		}
	}
}

func testContinueMultiChainLiveValidationFailure(t *testing.T) {
	t.Helper()
	env := newContinuationEnvForGameTypes(t, []embedded.GameType{
		embedded.GameTypeSuperPermissioned,
		embedded.GameTypeSuperCannonKona,
	})
	setPermissionlessContinuationInputs(env.preparedChains[1], 2)
	require.NoError(t, pipeline.WriteState(env.workdir, env.prepared))
	nonceBefore := pendingNonce(t, env)
	latest, err := env.l1Client.HeaderByNumber(env.ctx, nil)
	require.NoError(t, err)
	secondReceiptBlock := new(big.Int).Add(latest.Number, big.NewInt(2))
	setAnvilCode(t, env.l1Client, env.standardValidator, conditionalValidatorCode(secondReceiptBlock))

	err = deployer.Continue(env.ctx, env.config())
	require.ErrorContains(t, err, "live deployment validation failed")
	require.ErrorContains(t, err, "TEST-FAIL")
	require.Equal(t, nonceBefore+2, pendingNonce(t, env))
	checkpointed, err := pipeline.ReadState(env.workdir)
	require.NoError(t, err)
	require.Nil(t, checkpointed.AppliedIntent)
	first, err := checkpointed.Chain(env.intent.Chains[0].ID)
	require.NoError(t, err)
	second, err := checkpointed.Chain(env.intent.Chains[1].ID)
	require.NoError(t, err)
	require.NotNil(t, first.Continuation)
	require.NotNil(t, second.Continuation)

	setAnvilCode(t, env.l1Client, env.standardValidator, validatorResultCode(validValidatorResult))
	nonceAfterFailure := pendingNonce(t, env)
	require.NoError(t, deployer.Continue(env.ctx, env.config()))
	require.Equal(t, nonceAfterFailure, pendingNonce(t, env))
	retried, err := pipeline.ReadState(env.workdir)
	require.NoError(t, err)
	require.Nil(t, retried.AppliedIntent)
	for _, chain := range env.intent.Chains {
		retriedChain, chainErr := retried.Chain(chain.ID)
		require.NoError(t, chainErr)
		require.NotNil(t, retriedChain.Continuation)
	}
}

func testContinueMultiChainSequentialValidation(t *testing.T) {
	t.Helper()
	env := newContinuationEnvForGameTypes(t, []embedded.GameType{
		embedded.GameTypeSuperCannonKona,
		embedded.GameTypeSuperPermissioned,
	})
	setPermissionlessContinuationInputs(env.preparedChains[0], 1)
	require.NoError(t, pipeline.WriteState(env.workdir, env.prepared))
	nonceBefore := pendingNonce(t, env)
	latest, err := env.l1Client.HeaderByNumber(env.ctx, nil)
	require.NoError(t, err)
	firstReceiptBlock := new(big.Int).Add(latest.Number, big.NewInt(1))
	setAnvilCode(t, env.l1Client, env.standardValidator, conditionalValidatorCode(firstReceiptBlock))

	err = deployer.Continue(env.ctx, env.config())
	require.ErrorContains(t, err, "live deployment validation failed")
	require.Equal(t, nonceBefore+1, pendingNonce(t, env))
	checkpointed, err := pipeline.ReadState(env.workdir)
	require.NoError(t, err)
	first, err := checkpointed.Chain(env.intent.Chains[0].ID)
	require.NoError(t, err)
	require.True(t, checkpointed.IsChainDeployed(env.intent.Chains[0].ID))
	require.NotNil(t, first.Continuation)
	require.False(t, checkpointed.IsChainDeployed(env.intent.Chains[1].ID))
	require.Nil(t, checkpointed.AppliedIntent)

	setAnvilCode(t, env.l1Client, env.standardValidator, validatorResultCode(validValidatorResult))
	require.NoError(t, deployer.Continue(env.ctx, env.config()))
	require.Equal(t, nonceBefore+2, pendingNonce(t, env))
	completed, err := pipeline.ReadState(env.workdir)
	require.NoError(t, err)
	require.Nil(t, completed.AppliedIntent)
	for _, chain := range env.intent.Chains {
		completedChain, chainErr := completed.Chain(chain.ID)
		require.NoError(t, chainErr)
		require.NotNil(t, completedChain.Continuation)
	}
}

func testContinueRejectsMixedPermissionlessFamilies(t *testing.T) {
	t.Helper()
	env := newContinuationEnvForGameTypes(t, []embedded.GameType{
		embedded.GameTypeSuperPermissioned,
		embedded.GameTypeSuperPermissioned,
	})
	gameTypes := []embedded.GameType{
		embedded.GameTypeCannonKona,
		embedded.GameTypeSuperCannonKona,
	}
	for i, gameType := range gameTypes {
		setPermissionlessContinuationInputs(env.preparedChains[i], uint64(i+1))
		env.intent.Chains[i].DeployOverrides = map[string]any{"respectedGameType": gameType}
		env.prepared.PreparedDeployment.Intent.Chains[i].DeployOverrides = map[string]any{"respectedGameType": gameType}
		recordedGameType := uint32(gameType)
		env.preparedChains[i].InitialGameType = &recordedGameType
	}
	require.NoError(t, env.intent.WriteToFile(filepath.Join(env.workdir, "intent.toml")))
	require.NoError(t, pipeline.WriteState(env.workdir, env.prepared))
	nonceBefore := pendingNonce(t, env)

	err := deployer.Continue(env.ctx, env.config())
	require.ErrorContains(t, err, "cannot mix CANNON_KONA and SUPER_CANNON_KONA")
	require.Equal(t, nonceBefore, pendingNonce(t, env))
}

func setPermissionlessContinuationInputs(chain *state.ChainState, sequenceNumber uint64) {
	chain.Prestate = common.BigToHash(new(big.Int).SetUint64(0x1200 + sequenceNumber))
	chain.StartingAnchorRoot = &state.StartingAnchorProposal{
		Root:             common.BigToHash(new(big.Int).SetUint64(0x5600 + sequenceNumber)),
		L2SequenceNumber: hexutil.Uint64(sequenceNumber),
	}
}

func newContinuationEnv(t *testing.T, gameType embedded.GameType) *continuationEnv {
	return newContinuationEnvForGameTypes(t, []embedded.GameType{gameType})
}

func newContinuationEnvForGameTypes(t *testing.T, gameTypes []embedded.GameType) *continuationEnv {
	return newContinuationEnvForGameTypesAndFeatures(t, gameTypes, common.Hash{})
}

func newContinuationEnvForGameTypesAndFeatures(
	t *testing.T,
	gameTypes []embedded.GameType,
	devFeatureBitmap common.Hash,
) *continuationEnv {
	return newContinuationEnvWithIntentMutator(t, gameTypes, devFeatureBitmap, nil)
}

func newContinuationEnvWithIntentMutator(
	t *testing.T,
	gameTypes []embedded.GameType,
	devFeatureBitmap common.Hash,
	mutateIntentAfterBootstrap func(*state.Intent),
) *continuationEnv {
	t.Helper()
	require.NotEmpty(t, gameTypes)
	ctx, cancel := context.WithTimeout(t.Context(), 180*time.Second)
	t.Cleanup(cancel)
	lgr, logs := testlog.CaptureLogger(t, slog.LevelWarn)
	l1RPC, l1Client := devnet.DefaultAnvilRPC(t, lgr)
	privateKey, key, dk := shared.DefaultPrivkey(t)
	l1ChainID := new(big.Int).SetUint64(devnet.DefaultChainID)
	loc, _ := testutil.LocalArtifacts(t)
	cacheDir := testutils.IsolatedTestDirWithAutoCleanup(t)
	intent, st := shared.NewIntent(t, l1ChainID, dk, uint256.NewInt(1), loc, loc, testCustomGasLimit)
	outputRootBootstrap := devfeatures.IsDevFeatureEnabled(devFeatureBitmap, devfeatures.OutputRootGamesFlag)
	intent.OutputRootBootstrap = outputRootBootstrap
	for i := 1; i < len(gameTypes); i++ {
		intent.Chains = append(
			intent.Chains,
			shared.NewChainIntent(t, dk, l1ChainID, uint256.NewInt(uint64(i+1)), testCustomGasLimit),
		)
	}

	superchainPAO := shared.AddrFor(t, dk, devkeys.L1ProxyAdminOwnerRole.Key(l1ChainID))
	bstrap, err := bootstrap.Superchain(ctx, bootstrap.SuperchainConfig{
		L1RPCUrl:                  l1RPC,
		PrivateKey:                privateKey,
		Logger:                    lgr,
		ArtifactsLocator:          loc,
		CacheDir:                  cacheDir,
		SuperchainProxyAdminOwner: superchainPAO,
		Guardian:                  intent.SuperchainRoles.SuperchainGuardian,
	})
	require.NoError(t, err)

	impls, err := bootstrap.Implementations(ctx, bootstrap.ImplementationsConfig{
		L1RPCUrl:                        l1RPC,
		PrivateKey:                      privateKey,
		ArtifactsLocator:                loc,
		MIPSVersion:                     int(standard.MIPSVersion),
		WithdrawalDelaySeconds:          standard.WithdrawalDelaySeconds,
		MinProposalSizeBytes:            standard.MinProposalSizeBytes,
		ChallengePeriodSeconds:          standard.ChallengePeriodSeconds,
		ProofMaturityDelaySeconds:       standard.ProofMaturityDelaySeconds,
		DisputeGameFinalityDelaySeconds: standard.DisputeGameFinalityDelaySeconds,
		DevFeatureBitmap:                devFeatureBitmap,
		OutputRootBootstrap:             outputRootBootstrap,
		SuperchainConfigProxy:           bstrap.SuperchainConfigProxy,
		L1ProxyAdminOwner:               intent.Chains[0].Roles.L1ProxyAdminOwner,
		SuperchainProxyAdmin:            bstrap.SuperchainProxyAdmin,
		CacheDir:                        cacheDir,
		Logger:                          lgr,
		Challenger:                      intent.Chains[0].Roles.Challenger,
		FaultGameMaxGameDepth:           standard.DisputeMaxGameDepth,
		FaultGameSplitDepth:             standard.DisputeSplitDepth,
		FaultGameClockExtension:         standard.DisputeClockExtension,
		FaultGameMaxClockDuration:       standard.DisputeMaxClockDuration,
	})
	require.NoError(t, err)

	if mutateIntentAfterBootstrap != nil {
		mutateIntentAfterBootstrap(intent)
	}
	intent.SuperchainRoles = nil
	intent.OPCMAddress = &impls.OpcmV2
	intent.SuperchainConfigProxy = &bstrap.SuperchainConfigProxy
	for i, gameType := range gameTypes {
		if intent.Chains[i].DeployOverrides == nil {
			intent.Chains[i].DeployOverrides = make(map[string]any)
		}
		intent.Chains[i].DeployOverrides["respectedGameType"] = gameType
	}
	workdir := t.TempDir()
	require.NoError(t, intent.WriteToFile(filepath.Join(workdir, "intent.toml")))
	require.NoError(t, pipeline.WriteState(workdir, st))
	require.NoError(t, deployer.Prepare(ctx, deployer.PrepareConfig{
		Workdir:           workdir,
		Logger:            lgr,
		PrivateKey:        privateKey,
		L1RPCUrl:          l1RPC,
		CacheDir:          cacheDir,
		GenesisTimeOffset: 600,
	}))

	prepared, err := pipeline.ReadState(workdir)
	require.NoError(t, err)
	require.NotNil(t, prepared.PreparedDeployment)
	preparedChains := make([]*state.ChainState, 0, len(intent.Chains))
	preparedSnapshotChains := make([]*state.PreparedChainState, 0, len(intent.Chains))
	for _, chain := range intent.Chains {
		require.False(t, prepared.IsChainDeployed(chain.ID))
		preparedChain, chainErr := prepared.Chain(chain.ID)
		require.NoError(t, chainErr)
		preparedChains = append(preparedChains, preparedChain)
		preparedSnapshotChain, chainErr := prepared.PreparedDeployment.Chain(chain.ID)
		require.NoError(t, chainErr)
		preparedSnapshotChains = append(preparedSnapshotChains, preparedSnapshotChain)
	}
	validator, err := opcm.NewContract(impls.OpcmV2, l1Client).OPCMStandardValidator(ctx)
	require.NoError(t, err)

	return &continuationEnv{
		ctx:                    ctx,
		lgr:                    lgr,
		logs:                   logs,
		l1RPC:                  l1RPC,
		l1Client:               l1Client,
		privateKey:             privateKey,
		deployer:               crypto.PubkeyToAddress(key.PublicKey),
		cacheDir:               cacheDir,
		workdir:                workdir,
		intent:                 intent,
		prepared:               prepared,
		preparedChain:          preparedChains[0],
		preparedChains:         preparedChains,
		preparedSnapshotChain:  preparedSnapshotChains[0],
		preparedSnapshotChains: preparedSnapshotChains,
		opcm:                   impls.OpcmV2,
		standardValidator:      validator,
	}
}

func (e *continuationEnv) config() deployer.ContinueConfig {
	return deployer.ContinueConfig{
		Workdir:    e.workdir,
		L1RPCUrl:   e.l1RPC,
		PrivateKey: e.privateKey,
		CacheDir:   e.cacheDir,
		Logger:     e.lgr,
	}
}

func pendingNonce(t *testing.T, env *continuationEnv) uint64 {
	t.Helper()
	nonce, err := env.l1Client.PendingNonceAt(env.ctx, env.deployer)
	require.NoError(t, err)
	return nonce
}

func assertContinuationCompleted(t *testing.T, env *continuationEnv, nonceBefore uint64) *state.State {
	t.Helper()
	require.Equal(t, nonceBefore+1, pendingNonce(t, env))
	continued, err := pipeline.ReadState(env.workdir)
	require.NoError(t, err)
	require.True(t, continued.IsChainDeployed(env.intent.Chains[0].ID))
	require.Nil(t, continued.AppliedIntent)
	continuedChain, err := continued.Chain(env.intent.Chains[0].ID)
	require.NoError(t, err)
	require.NotNil(t, continuedChain.Continuation)
	continuedSnapshot, err := continued.PreparedDeployment.Chain(env.intent.Chains[0].ID)
	require.NoError(t, err)
	require.Equal(t, env.preparedSnapshotChain.OpChainContracts, continuedSnapshot.OpChainContracts)
	return continued
}

func setAnvilCode(t *testing.T, client *ethclient.Client, address common.Address, code []byte) {
	t.Helper()
	require.NoError(t, client.Client().Call(nil, "anvil_setCode", address, hexutil.Encode(code)))
}

const validValidatorResult = "OVERRIDES-L1PAOMULTISIG,OVERRIDES-CHALLENGER"

func validatorResultCode(result string) []byte {
	code := []byte{
		byte(vm.PUSH1), 0x20, byte(vm.PUSH1), 0, byte(vm.MSTORE),
		byte(vm.PUSH1), byte(len(result)), byte(vm.PUSH1), 0x20, byte(vm.MSTORE),
	}
	data := []byte(result)
	for offset := 0; offset < len(data); offset += 32 {
		end := offset + 32
		if end > len(data) {
			end = len(data)
		}
		code = append(code, byte(vm.PUSH32))
		code = append(code, common.RightPadBytes(data[offset:end], 32)...)
		code = append(code, byte(vm.PUSH1), byte(0x40+offset), byte(vm.MSTORE))
	}
	returnSize := byte(0x40 + 32*((len(data)+31)/32))
	return append(code, byte(vm.PUSH1), returnSize, byte(vm.PUSH1), 0, byte(vm.RETURN))
}

func conditionalValidatorCode(liveValidationBlock *big.Int) []byte {
	code := []byte{byte(vm.NUMBER), byte(vm.PUSH32)}
	code = append(code, common.LeftPadBytes(liveValidationBlock.Bytes(), 32)...)
	code = append(code, byte(vm.EQ), byte(vm.PUSH1), 0, byte(vm.JUMPI))
	jumpDestinationIndex := len(code) - 2
	code = append(code, validatorResultCode(validValidatorResult)...)
	code[jumpDestinationIndex] = byte(len(code))
	code = append(code, byte(vm.JUMPDEST))
	return append(code, validatorResultCode("TEST-FAIL")...)
}
