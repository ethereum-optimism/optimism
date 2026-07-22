package integration_test

import (
	"context"
	"log/slog"
	"math/big"
	"path/filepath"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
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
	ctx               context.Context
	lgr               log.Logger
	logs              *testlog.CapturingHandler
	l1RPC             string
	l1Client          *ethclient.Client
	privateKey        string
	deployer          common.Address
	cacheDir          string
	workdir           string
	intent            *state.Intent
	prepared          *state.State
	preparedChain     *state.ChainState
	opcm              common.Address
	standardValidator common.Address
}

func TestEndToEndContinuePreparedChain(t *testing.T) {
	op_e2e.InitParallel(t)

	t.Run("permissionless", func(t *testing.T) {
		testContinuePermissionless(t)
	})
	t.Run("permissioned without committed prestate", func(t *testing.T) {
		testContinuePermissioned(t)
	})
	t.Run("live validation failure persists checkpoint", func(t *testing.T) {
		testContinueLiveValidationFailure(t)
	})
	t.Run("partial deployment is rejected", func(t *testing.T) {
		testContinuePartialDeployment(t)
	})
}

func testContinuePermissionless(t *testing.T) {
	t.Helper()
	env := newContinuationEnv(t, embedded.GameTypeCannonKona)
	env.preparedChain.Prestate = common.HexToHash("0x1234")
	env.preparedChain.StartingAnchorRoot = &state.StartingAnchorProposal{
		Root:             common.HexToHash("0x5678"),
		L2SequenceNumber: 7,
	}
	originalAnchor := env.preparedChain.StartBlock.Hash
	originalContracts := env.preparedChain.OpChainContracts
	require.NoError(t, pipeline.WriteState(env.workdir, env.prepared))
	nonceBefore := pendingNonce(t, env)

	env.preparedChain.StartBlock.Hash = common.HexToHash("0xdead")
	require.NoError(t, pipeline.WriteState(env.workdir, env.prepared))
	err := deployer.Continue(env.ctx, env.config())
	require.ErrorContains(t, err, "pinned anchor block")
	require.Equal(t, nonceBefore, pendingNonce(t, env))
	env.preparedChain.StartBlock.Hash = originalAnchor

	validatorCode, err := env.l1Client.CodeAt(env.ctx, env.standardValidator, nil)
	require.NoError(t, err)
	setAnvilCode(t, env.l1Client, env.standardValidator, validatorResultCode("TEST-FAIL"))
	require.NoError(t, pipeline.WriteState(env.workdir, env.prepared))
	err = deployer.Continue(env.ctx, env.config())
	require.ErrorContains(t, err, "standard validator reported errors: TEST-FAIL")
	require.Equal(t, nonceBefore, pendingNonce(t, env))
	setAnvilCode(t, env.l1Client, env.standardValidator, validatorCode)

	opcmCode, err := env.l1Client.CodeAt(env.ctx, env.opcm, nil)
	require.NoError(t, err)
	setAnvilCode(t, env.l1Client, env.opcm, []byte{byte(vm.PUSH0), byte(vm.PUSH0), byte(vm.REVERT)})
	require.NoError(t, pipeline.WriteState(env.workdir, env.prepared))
	err = deployer.Continue(env.ctx, env.config())
	require.ErrorContains(t, err, "continuation preflight failed")
	require.Equal(t, nonceBefore, pendingNonce(t, env))
	setAnvilCode(t, env.l1Client, env.opcm, opcmCode)

	env.preparedChain.SystemConfigProxy = common.Address{0xff}
	require.NoError(t, pipeline.WriteState(env.workdir, env.prepared))
	err = deployer.Continue(env.ctx, env.config())
	require.ErrorContains(t, err, "simulated contract addresses differ")
	require.Equal(t, nonceBefore, pendingNonce(t, env))

	env.preparedChain.OpChainContracts = originalContracts
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
	require.Equal(t, originalContracts, *reconciledChain.Continuation.ExpectedContracts)
	require.Nil(t, reconciledChain.Continuation.TxHash)
	require.Nil(t, reconciledChain.Continuation.ReceiptBlockNumber)
	require.Nil(t, reconciledChain.Continuation.ReceiptBlockHash)
	require.True(t, reconciledChain.Continuation.LiveValidated)
	require.NotNil(t, reconciled.AppliedIntent)
}

func testContinuePermissioned(t *testing.T) {
	t.Helper()
	env := newContinuationEnv(t, embedded.GameTypePermissionedCannon)
	require.Zero(t, env.preparedChain.Prestate)
	elapsedGenesisTime := hexutil.Uint64(1)
	env.preparedChain.GenesisTime = &elapsedGenesisTime
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
	require.ErrorContains(t, err, "live deployment validation failed for deployed chain")
	require.ErrorContains(t, err, "SystemConfigProxy code")
	require.Equal(t, nonceAfter, pendingNonce(t, env))
	stale, err := pipeline.ReadState(env.workdir)
	require.NoError(t, err)
	staleChain, err := stale.Chain(env.intent.Chains[0].ID)
	require.NoError(t, err)
	require.False(t, staleChain.Continuation.LiveValidated)
}

func testContinueLiveValidationFailure(t *testing.T) {
	t.Helper()
	env := newContinuationEnv(t, embedded.GameTypeCannonKona)
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
	require.Equal(t, env.preparedChain.OpChainContracts, *continuedChain.Continuation.ExpectedContracts)
	require.NotNil(t, continuedChain.Continuation.TxHash)
	require.NotNil(t, continuedChain.Continuation.ReceiptBlockNumber)
	require.NotNil(t, continuedChain.Continuation.ReceiptBlockHash)
	require.False(t, continuedChain.Continuation.LiveValidated)

	setAnvilCode(t, env.l1Client, env.standardValidator, validatorResultCode(""))
	nonceAfterFailure := pendingNonce(t, env)
	require.NoError(t, deployer.Continue(env.ctx, env.config()))
	require.Equal(t, nonceAfterFailure, pendingNonce(t, env))
	retried, err := pipeline.ReadState(env.workdir)
	require.NoError(t, err)
	retriedChain, err := retried.Chain(env.intent.Chains[0].ID)
	require.NoError(t, err)
	require.True(t, retriedChain.Continuation.LiveValidated)
	require.Equal(t, continuedChain.Continuation.TxHash, retriedChain.Continuation.TxHash)
	require.Equal(t, continuedChain.Continuation.ReceiptBlockNumber, retriedChain.Continuation.ReceiptBlockNumber)
	require.Equal(t, continuedChain.Continuation.ReceiptBlockHash, retriedChain.Continuation.ReceiptBlockHash)
	require.NotNil(t, retried.AppliedIntent)
}

func testContinuePartialDeployment(t *testing.T) {
	t.Helper()
	env := newContinuationEnv(t, embedded.GameTypeCannonKona)
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

func newContinuationEnv(t *testing.T, gameType embedded.GameType) *continuationEnv {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 90*time.Second)
	t.Cleanup(cancel)
	lgr, logs := testlog.CaptureLogger(t, slog.LevelWarn)
	l1RPC, l1Client := devnet.DefaultAnvilRPC(t, lgr)
	privateKey, key, dk := shared.DefaultPrivkey(t)
	l1ChainID := new(big.Int).SetUint64(devnet.DefaultChainID)
	loc, _ := testutil.LocalArtifacts(t)
	cacheDir := testutils.IsolatedTestDirWithAutoCleanup(t)
	intent, st := shared.NewIntent(t, l1ChainID, dk, uint256.NewInt(1), loc, loc, testCustomGasLimit)

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

	intent.SuperchainRoles = nil
	intent.OPCMAddress = &impls.OpcmV2
	intent.SuperchainConfigProxy = &bstrap.SuperchainConfigProxy
	intent.Chains[0].DeployOverrides = map[string]any{"respectedGameType": gameType}
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
	require.True(t, prepared.Prepared)
	require.False(t, prepared.IsChainDeployed(intent.Chains[0].ID))
	preparedChain, err := prepared.Chain(intent.Chains[0].ID)
	require.NoError(t, err)
	validator, err := opcm.NewContract(impls.OpcmV2, l1Client).OPCMStandardValidator(ctx)
	require.NoError(t, err)

	return &continuationEnv{
		ctx:               ctx,
		lgr:               lgr,
		logs:              logs,
		l1RPC:             l1RPC,
		l1Client:          l1Client,
		privateKey:        privateKey,
		deployer:          crypto.PubkeyToAddress(key.PublicKey),
		cacheDir:          cacheDir,
		workdir:           workdir,
		intent:            intent,
		prepared:          prepared,
		preparedChain:     preparedChain,
		opcm:              impls.OpcmV2,
		standardValidator: validator,
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
	require.NotNil(t, continued.AppliedIntent)
	continuedChain, err := continued.Chain(env.intent.Chains[0].ID)
	require.NoError(t, err)
	require.NotNil(t, continuedChain.Continuation)
	require.Equal(t, env.preparedChain.OpChainContracts, *continuedChain.Continuation.ExpectedContracts)
	require.NotNil(t, continuedChain.Continuation.TxHash)
	require.NotNil(t, continuedChain.Continuation.ReceiptBlockNumber)
	require.NotNil(t, continuedChain.Continuation.ReceiptBlockHash)
	require.True(t, continuedChain.Continuation.LiveValidated)
	return continued
}

func setAnvilCode(t *testing.T, client *ethclient.Client, address common.Address, code []byte) {
	t.Helper()
	require.NoError(t, client.Client().Call(nil, "anvil_setCode", address, hexutil.Encode(code)))
}

func validatorResultCode(result string) []byte {
	data := common.RightPadBytes([]byte(result), 32)
	code := []byte{
		byte(vm.PUSH1), 0x20, byte(vm.PUSH1), 0, byte(vm.MSTORE),
		byte(vm.PUSH1), byte(len(result)), byte(vm.PUSH1), 0x20, byte(vm.MSTORE),
		byte(vm.PUSH32),
	}
	code = append(code, data...)
	code = append(code, byte(vm.PUSH1), 0x40, byte(vm.MSTORE))
	returnSize := byte(0x40)
	if result != "" {
		returnSize = 0x60
	}
	return append(code, byte(vm.PUSH1), returnSize, byte(vm.PUSH1), 0, byte(vm.RETURN))
}

func conditionalValidatorCode(liveValidationBlock *big.Int) []byte {
	code := []byte{byte(vm.NUMBER), byte(vm.PUSH32)}
	code = append(code, common.LeftPadBytes(liveValidationBlock.Bytes(), 32)...)
	code = append(code, byte(vm.EQ), byte(vm.PUSH1), 0, byte(vm.JUMPI))
	jumpDestinationIndex := len(code) - 2
	code = append(code, validatorResultCode("")...)
	code[jumpDestinationIndex] = byte(len(code))
	code = append(code, byte(vm.JUMPDEST))
	return append(code, validatorResultCode("TEST-FAIL")...)
}
