package cli

import (
	"context"
	"log/slog"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/bootstrap"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/integration_test/shared"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/pipeline"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/standard"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/testutil"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/upgrade/embedded"
	op_e2e "github.com/ethereum-optimism/optimism/op-e2e"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils/devnet"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

const continueCLIGasLimit = uint64(90_123_456)

type continueCLIFixture struct {
	preparedWorkdir string
	l1RPC           string
	privateKey      string
	deployer        common.Address
	l1Client        *ethclient.Client
	chainID         common.Hash
}

func TestContinueCLIColdStart(t *testing.T) {
	op_e2e.InitParallel(t)
	fixture := newContinueCLIFixture(t)

	t.Run("use-forge is rejected by the parser", func(t *testing.T) {
		workdir := copyContinueCLIWorkdir(t, fixture.preparedWorkdir)
		runner := NewCLITestRunner(t)
		nonceBefore := continueCLINonce(t, fixture)

		runner.ExpectErrorContains(t, append(
			continueCLIArgs(fixture, workdir),
			"--use-forge",
		), nil, "flag provided but not defined: -use-forge")

		require.Equal(t, nonceBefore, continueCLINonce(t, fixture))
	})

	t.Run("frozen value changed on disk is rejected", func(t *testing.T) {
		workdir := copyContinueCLIWorkdir(t, fixture.preparedWorkdir)
		intent, err := pipeline.ReadIntent(workdir)
		require.NoError(t, err)
		changedOPCM := common.Address{0xaa}
		intent.OPCMAddress = &changedOPCM
		require.NoError(t, intent.WriteToFile(filepath.Join(workdir, "intent.toml")))

		runner := NewCLITestRunner(t)
		nonceBefore := continueCLINonce(t, fixture)
		runner.ExpectErrorContains(
			t,
			continueCLIArgs(fixture, workdir),
			nil,
			"intent OPCM address changed after prepare",
		)

		require.Equal(t, nonceBefore, continueCLINonce(t, fixture))
		persisted, err := pipeline.ReadState(workdir)
		require.NoError(t, err)
		require.False(t, persisted.IsChainDeployed(fixture.chainID))
		require.Nil(t, persisted.AppliedIntent)
	})

	t.Run("committed workdir and fresh cache are sufficient", func(t *testing.T) {
		workdir := copyContinueCLIWorkdir(t, fixture.preparedWorkdir)
		prepared, err := pipeline.ReadState(workdir)
		require.NoError(t, err)
		preparedChain, err := prepared.Chain(fixture.chainID)
		require.NoError(t, err)
		expectedContracts := preparedChain.OpChainContracts

		runner := NewCLITestRunner(t)
		require.NoDirExists(t, filepath.Join(runner.GetWorkDir(), ".cache"))
		nonceBefore := continueCLINonce(t, fixture)
		runner.ExpectSuccess(t, continueCLIArgs(fixture, workdir), nil)

		require.Equal(t, nonceBefore+1, continueCLINonce(t, fixture))
		persisted, err := pipeline.ReadState(workdir)
		require.NoError(t, err)
		require.Nil(t, persisted.AppliedIntent)
		require.True(t, persisted.IsChainDeployed(fixture.chainID))
		continuedChain, err := persisted.Chain(fixture.chainID)
		require.NoError(t, err)
		require.NotNil(t, continuedChain.Continuation)
		continuedSnapshot, err := persisted.PreparedDeployment.Chain(fixture.chainID)
		require.NoError(t, err)
		require.Equal(t, expectedContracts, continuedSnapshot.OpChainContracts)

		code, err := fixture.l1Client.CodeAt(t.Context(), continuedChain.SystemConfigProxy, nil)
		require.NoError(t, err)
		require.NotEmpty(t, code)
	})
}

func newContinueCLIFixture(t *testing.T) *continueCLIFixture {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 180*time.Second)
	defer cancel()
	lgr := testlog.Logger(t, slog.LevelError)
	l1RPC, l1Client := devnet.DefaultAnvilRPC(t, lgr)
	t.Cleanup(l1Client.Close)
	privateKey, key, dk := shared.DefaultPrivkey(t)
	l1ChainID := new(big.Int).SetUint64(devnet.DefaultChainID)
	chainID := uint256.NewInt(1)
	loc, _ := testutil.LocalArtifacts(t)
	setupCacheDir := t.TempDir()

	intent, st := shared.NewIntent(t, l1ChainID, dk, chainID, loc, loc, continueCLIGasLimit)
	superchainPAO := shared.AddrFor(t, dk, devkeys.L1ProxyAdminOwnerRole.Key(l1ChainID))
	bstrap, err := bootstrap.Superchain(ctx, bootstrap.SuperchainConfig{
		L1RPCUrl:                  l1RPC,
		PrivateKey:                privateKey,
		Logger:                    lgr,
		ArtifactsLocator:          loc,
		CacheDir:                  setupCacheDir,
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
		CacheDir:                        setupCacheDir,
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
	intent.Chains[0].DeployOverrides = map[string]any{"respectedGameType": embedded.GameTypeSuperCannonKona}
	preparedWorkdir := t.TempDir()
	require.NoError(t, intent.WriteToFile(filepath.Join(preparedWorkdir, "intent.toml")))
	require.NoError(t, pipeline.WriteState(preparedWorkdir, st))
	require.NoError(t, deployer.Prepare(ctx, deployer.PrepareConfig{
		Workdir:           preparedWorkdir,
		Logger:            lgr,
		PrivateKey:        privateKey,
		L1RPCUrl:          l1RPC,
		CacheDir:          setupCacheDir,
		GenesisTimeOffset: 600,
	}))

	prepared, err := pipeline.ReadState(preparedWorkdir)
	require.NoError(t, err)
	preparedChain, err := prepared.Chain(chainID.Bytes32())
	require.NoError(t, err)
	preparedChain.Prestate = common.HexToHash("0x1234")
	preparedChain.StartingAnchorRoot = &state.StartingAnchorProposal{
		Root:             common.HexToHash("0x5678"),
		L2SequenceNumber: 7,
	}
	require.NoError(t, pipeline.WriteState(preparedWorkdir, prepared))

	return &continueCLIFixture{
		preparedWorkdir: preparedWorkdir,
		l1RPC:           l1RPC,
		privateKey:      privateKey,
		deployer:        crypto.PubkeyToAddress(key.PublicKey),
		l1Client:        l1Client,
		chainID:         chainID.Bytes32(),
	}
}

func copyContinueCLIWorkdir(t *testing.T, source string) string {
	t.Helper()
	destination := filepath.Join(t.TempDir(), "workdir")
	require.NoError(t, os.CopyFS(destination, os.DirFS(source)))
	return destination
}

func continueCLIArgs(fixture *continueCLIFixture, workdir string) []string {
	return []string{
		"continue",
		"--workdir", workdir,
		"--l1-rpc-url", fixture.l1RPC,
		"--private-key", fixture.privateKey,
	}
}

func continueCLINonce(t *testing.T, fixture *continueCLIFixture) uint64 {
	t.Helper()
	nonce, err := fixture.l1Client.PendingNonceAt(t.Context(), fixture.deployer)
	require.NoError(t, err)
	return nonce
}
