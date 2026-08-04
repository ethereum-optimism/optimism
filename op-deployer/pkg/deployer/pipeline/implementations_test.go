package pipeline

import (
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/addresses"
	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum-optimism/optimism/op-core/devfeatures"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/artifacts"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestGenesisMockSP1VerifierArtifactAndDeployment(t *testing.T) {
	artifactsFS, err := artifacts.Download(context.Background(), artifacts.EmbeddedLocator, nil, t.TempDir())
	require.NoError(t, err)
	af := &foundry.ArtifactsFS{FS: artifactsFS}
	artifact, err := af.ReadArtifact(genesisMockSP1VerifierArtifact, genesisMockSP1VerifierContract)
	require.NoError(t, err)
	require.NotEmpty(t, artifact.Bytecode.Object, "genesis MockSP1Verifier must have bytecode in op-deployer artifacts")

	host := script.NewHost(testlog.Logger(t, slog.LevelDebug), af, nil, script.DefaultContext)
	intent := new(state.Intent)
	verifier, err := deployAndRecordGenesisMockSP1Verifier(&Env{
		L1ScriptHost: host,
		Deployer:     common.Address{0xdd},
	}, intent)
	require.NoError(t, err)
	require.NotEqual(t, common.Address{}, verifier)
	require.NotEmpty(t, host.GetCode(verifier))
	require.Equal(t, verifier, intent.GlobalDeployOverrides["sp1Verifier"])
}

func TestDeployImplementationsSP1VerifierValidation(t *testing.T) {
	newState := func() *state.State {
		return &state.State{
			SuperchainDeployment: &addresses.SuperchainContracts{
				SuperchainConfigProxy:    common.Address{0x01},
				SuperchainProxyAdminImpl: common.Address{0x02},
			},
			SuperchainRoles: &addresses.SuperchainRoles{
				SuperchainProxyAdminOwner: common.Address{0x03},
				Challenger:                common.Address{0x04},
			},
		}
	}
	env := &Env{Logger: testlog.Logger(t, slog.LevelDebug)}

	t.Run("disabled rejects verifier", func(t *testing.T) {
		intent := &state.Intent{GlobalDeployOverrides: map[string]any{
			"sp1Verifier": common.Address{0x05},
		}}
		err := DeployImplementations(env, intent, newState())
		require.ErrorContains(t, err, "must not be specified")
	})

	t.Run("live enabled requires verifier", func(t *testing.T) {
		intent := &state.Intent{GlobalDeployOverrides: map[string]any{
			"devFeatureBitmap": devfeatures.ZKDisputeGameFlag,
		}}
		err := DeployImplementations(env, intent, newState())
		require.ErrorContains(t, err, "must be specified")
	})

	t.Run("genesis enabled still requires an explicit verifier selection", func(t *testing.T) {
		intent := &state.Intent{GlobalDeployOverrides: map[string]any{
			"devFeatureBitmap": devfeatures.ZKDisputeGameFlag,
		}}
		genesisEnv := *env
		genesisEnv.IsGenesis = true
		genesisEnv.AllowUnoptimizedContracts = true
		err := DeployImplementations(&genesisEnv, intent, newState())
		require.ErrorContains(t, err, "sp1Verifier must be specified")
	})

	t.Run("explicit mock selection requires a script host", func(t *testing.T) {
		intent := &state.Intent{GlobalDeployOverrides: map[string]any{
			"devFeatureBitmap": devfeatures.ZKDisputeGameFlag,
		}}
		mockEnv := *env
		mockEnv.IsGenesis = true
		mockEnv.DeployMockSP1Verifier = true
		err := DeployImplementations(&mockEnv, intent, newState())
		require.ErrorContains(t, err, "without an L1 script host")
	})

	t.Run("explicit zero verifier is rejected", func(t *testing.T) {
		intent := &state.Intent{GlobalDeployOverrides: map[string]any{
			"devFeatureBitmap": devfeatures.ZKDisputeGameFlag,
			"sp1Verifier":      common.Address{},
		}}
		err := DeployImplementations(env, intent, newState())
		require.ErrorContains(t, err, "sp1Verifier override must not be zero")
	})

	t.Run("predeployed OPCM rejects verifier override", func(t *testing.T) {
		opcmAddress := common.Address{0x06}
		intent := &state.Intent{
			OPCMAddress: &opcmAddress,
			GlobalDeployOverrides: map[string]any{
				"sp1Verifier": common.Address{0x05},
			},
		}
		st := newState()
		st.ImplementationsDeployment = &addresses.ImplementationsContracts{OpcmV2Impl: opcmAddress}
		err := DeployImplementations(env, intent, st)
		require.ErrorContains(t, err, "must not be specified when using a predeployed OPCM")
	})

	t.Run("reused implementations require the same verifier", func(t *testing.T) {
		artifactsFS, err := artifacts.Download(context.Background(), artifacts.EmbeddedLocator, nil, t.TempDir())
		require.NoError(t, err)
		af := &foundry.ArtifactsFS{FS: artifactsFS}
		host := script.NewHost(testlog.Logger(t, slog.LevelDebug), af, nil, script.DefaultContext)
		adapterArtifact, err := af.ReadArtifact("SP1PlonkAdapter.sol", "SP1PlonkAdapter")
		require.NoError(t, err)

		rawVerifier, err := deployGenesisMockSP1Verifier(&Env{
			L1ScriptHost: host,
			Deployer:     common.Address{0xdd},
		})
		require.NoError(t, err)
		constructorArgs, err := adapterArtifact.ABI.Pack("", rawVerifier)
		require.NoError(t, err)
		adapter, err := host.Create(common.Address{0xdd}, append(adapterArtifact.Bytecode.Object, constructorArgs...))
		require.NoError(t, err)

		st := newState()
		st.ImplementationsDeployment = &addresses.ImplementationsContracts{SP1PlonkAdapterImpl: adapter}
		st.SP1Verifier = &rawVerifier
		resumeEnv := *env
		resumeEnv.L1ScriptHost = host

		// Use alternate casing to ensure bootstrap-only overrides cannot bypass reuse validation.
		sameIntent := &state.Intent{GlobalDeployOverrides: map[string]any{"SP1Verifier": rawVerifier}}
		require.NoError(t, DeployImplementations(&resumeEnv, sameIntent, st))

		restartedGenesisEnv := *env
		restartedGenesisEnv.IsGenesis = true
		restartedGenesisEnv.DeployMockSP1Verifier = true
		checkpoint, err := json.Marshal(st)
		require.NoError(t, err)
		var resumedState state.State
		require.NoError(t, json.Unmarshal(checkpoint, &resumedState))
		explicitGenesisEnv := *env
		explicitGenesisEnv.IsGenesis = true
		explicitGenesisIntent := &state.Intent{GlobalDeployOverrides: map[string]any{
			"sp1Verifier": rawVerifier,
		}}
		require.NoError(t, DeployImplementations(&explicitGenesisEnv, explicitGenesisIntent, &resumedState))

		unselectedGenesisEnv := *env
		unselectedGenesisEnv.IsGenesis = true
		unselectedGenesisIntent := &state.Intent{GlobalDeployOverrides: map[string]any{
			"devFeatureBitmap": devfeatures.ZKDisputeGameFlag,
		}}
		err = DeployImplementations(&unselectedGenesisEnv, unselectedGenesisIntent, &resumedState)
		require.ErrorContains(t, err, "sp1Verifier must be specified")

		resumedGenesisIntent := &state.Intent{GlobalDeployOverrides: map[string]any{
			"devFeatureBitmap": devfeatures.ZKDisputeGameFlag,
		}}
		require.NoError(t, DeployImplementations(&restartedGenesisEnv, resumedGenesisIntent, &resumedState))
		require.Equal(t, rawVerifier, resumedGenesisIntent.GlobalDeployOverrides["sp1Verifier"])

		changedIntent := &state.Intent{GlobalDeployOverrides: map[string]any{"sp1Verifier": common.Address{0x08}}}
		err = DeployImplementations(&resumeEnv, changedIntent, st)
		require.ErrorContains(t, err, "does not match")
	})
}
