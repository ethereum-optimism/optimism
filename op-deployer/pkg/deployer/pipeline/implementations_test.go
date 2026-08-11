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
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
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
	verifier, err := deployGenesisMockSP1Verifier(&Env{
		L1ScriptHost: host,
		Deployer:     common.Address{0xdd},
	})
	require.NoError(t, err)
	require.NotEqual(t, common.Address{}, verifier)
	require.NotEmpty(t, host.GetCode(verifier))
}

func TestParseSP1VerifierOverrideRequiresExactKey(t *testing.T) {
	_, found, err := parseSP1VerifierOverride(map[string]any{"SP1Verifier": common.Address{0x05}})
	require.NoError(t, err)
	require.False(t, found)
}

func newSP1VerifierTestState() *state.State {
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

func TestDeployImplementationsSP1VerifierValidation(t *testing.T) {
	newState := newSP1VerifierTestState
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

		disabledIntent := &state.Intent{GlobalDeployOverrides: map[string]any{"sp1Verifier": rawVerifier}}
		err = DeployImplementations(&resumeEnv, disabledIntent, st)
		require.ErrorContains(t, err, "ZK dispute games are disabled")

		unselectedLiveIntent := &state.Intent{GlobalDeployOverrides: map[string]any{
			"devFeatureBitmap": devfeatures.ZKDisputeGameFlag,
		}}
		err = DeployImplementations(&resumeEnv, unselectedLiveIntent, st)
		require.ErrorContains(t, err, "sp1Verifier must be specified")

		sameIntent := &state.Intent{GlobalDeployOverrides: map[string]any{
			"devFeatureBitmap": devfeatures.ZKDisputeGameFlag,
			"sp1Verifier":      rawVerifier,
		}}
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
			"devFeatureBitmap": devfeatures.ZKDisputeGameFlag,
			"sp1Verifier":      rawVerifier,
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
		require.NotContains(t, resumedGenesisIntent.GlobalDeployOverrides, "sp1Verifier")

		changedIntent := &state.Intent{GlobalDeployOverrides: map[string]any{
			"devFeatureBitmap": devfeatures.ZKDisputeGameFlag,
			"sp1Verifier":      common.Address{0x08},
		}}
		err = DeployImplementations(&resumeEnv, changedIntent, st)
		require.ErrorContains(t, err, "does not match")
	})
}

// capturingDeployImplementations records the input DeployImplementations would broadcast. The
// embedded nil ForgeScript satisfies the rest of the interface; only Run is exercised.
type capturingDeployImplementations struct {
	script.ForgeScript
	input opcm.DeployImplementationsInput
	ran   bool
}

func (c *capturingDeployImplementations) Run(input opcm.DeployImplementationsInput) (opcm.DeployImplementationsOutput, error) {
	c.input = input
	c.ran = true
	var out opcm.DeployImplementationsOutput
	return out, nil
}

// TestDeployImplementationsSelectsStandardSP1Verifier pins which raw SP1 verifier a live
// ZK-enabled apply hands to DeployImplementations.
func TestDeployImplementationsSelectsStandardSP1Verifier(t *testing.T) {
	const standardVerifier = "0xc3c6dDDAc8829b233Dc6536Ec024775a57b0AF2A"

	deploy := func(t *testing.T, env Env, intent *state.Intent, st *state.State) (*capturingDeployImplementations, error) {
		capture := new(capturingDeployImplementations)
		env.Logger = testlog.Logger(t, slog.LevelDebug)
		env.Scripts = &opcm.Scripts{DeployImplementations: capture}
		return capture, DeployImplementations(&env, intent, st)
	}

	zkIntent := func(l1ChainID uint64, overrides map[string]any) *state.Intent {
		merged := map[string]any{"devFeatureBitmap": devfeatures.ZKDisputeGameFlag}
		for k, v := range overrides {
			merged[k] = v
		}
		return &state.Intent{L1ChainID: l1ChainID, GlobalDeployOverrides: merged}
	}

	for _, tt := range []struct {
		name      string
		l1ChainID uint64
	}{
		{"mainnet", 1},
		{"sepolia", 11155111},
	} {
		t.Run(tt.name+" defaults to the release verifier", func(t *testing.T) {
			st := newSP1VerifierTestState()
			intent := zkIntent(tt.l1ChainID, nil)
			capture, err := deploy(t, Env{}, intent, st)
			require.NoError(t, err)
			require.Equal(t, common.HexToAddress(standardVerifier), capture.input.SP1Verifier)
			require.NotNil(t, st.SP1Verifier)
			require.Equal(t, common.HexToAddress(standardVerifier), *st.SP1Verifier)
			require.NotContains(t, intent.GlobalDeployOverrides, "sp1Verifier")
		})
	}

	t.Run("explicit override wins", func(t *testing.T) {
		override := common.Address{0x05}
		capture, err := deploy(t, Env{}, zkIntent(11155111, map[string]any{"sp1Verifier": override}), newSP1VerifierTestState())
		require.NoError(t, err)
		require.Equal(t, override, capture.input.SP1Verifier)
	})

	t.Run("unsupported network requires an override", func(t *testing.T) {
		capture, err := deploy(t, Env{}, zkIntent(900, nil), newSP1VerifierTestState())
		require.ErrorContains(t, err, "sp1Verifier must be specified")
		require.ErrorContains(t, err, "900")
		require.False(t, capture.ran)
	})

	t.Run("disabled ZK selects nothing", func(t *testing.T) {
		st := newSP1VerifierTestState()
		capture, err := deploy(t, Env{}, &state.Intent{L1ChainID: 11155111}, st)
		require.NoError(t, err)
		require.Equal(t, common.Address{}, capture.input.SP1Verifier)
		require.Nil(t, st.SP1Verifier)
	})

	t.Run("genesis does not select the release verifier", func(t *testing.T) {
		capture, err := deploy(t, Env{IsGenesis: true}, zkIntent(11155111, nil), newSP1VerifierTestState())
		require.ErrorContains(t, err, "sp1Verifier must be specified")
		require.False(t, capture.ran)
	})

	t.Run("resumed deployment keeps the verifier recorded in state", func(t *testing.T) {
		recorded := common.Address{0x06}
		st := newSP1VerifierTestState()
		st.ImplementationsDeployment = &addresses.ImplementationsContracts{SP1PlonkAdapterImpl: common.Address{0xad}}
		st.SP1Verifier = &recorded

		capture, err := deploy(t, Env{}, zkIntent(11155111, nil), st)
		require.NoError(t, err)
		require.False(t, capture.ran)
		require.Equal(t, recorded, *st.SP1Verifier)
	})
}
