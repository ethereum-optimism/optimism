package pipeline

import (
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/addresses"
	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/artifacts"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

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

	t.Run("explicit zero verifier is rejected", func(t *testing.T) {
		intent := &state.Intent{GlobalDeployOverrides: map[string]any{
			"sp1Verifier": common.Address{},
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
		verifierArtifact, err := af.ReadArtifact("MockSP1Verifier.sol", "MockSP1Verifier")
		require.NoError(t, err)

		rawVerifier, err := host.Create(common.Address{0xdd}, verifierArtifact.Bytecode.Object)
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

		sameIntent := &state.Intent{GlobalDeployOverrides: map[string]any{"sp1Verifier": rawVerifier}}
		require.NoError(t, DeployImplementations(&resumeEnv, sameIntent, st))

		checkpoint, err := json.Marshal(st)
		require.NoError(t, err)
		var resumedState state.State
		require.NoError(t, json.Unmarshal(checkpoint, &resumedState))
		genesisEnv := *env
		genesisEnv.IsGenesis = true
		explicitGenesisIntent := &state.Intent{GlobalDeployOverrides: map[string]any{"sp1Verifier": rawVerifier}}
		require.NoError(t, DeployImplementations(&genesisEnv, explicitGenesisIntent, &resumedState))

		unselectedGenesisIntent := &state.Intent{GlobalDeployOverrides: map[string]any{}}
		require.NoError(t, DeployImplementations(&genesisEnv, unselectedGenesisIntent, &resumedState))
		require.NotContains(t, unselectedGenesisIntent.GlobalDeployOverrides, "sp1Verifier")

		changedIntent := &state.Intent{GlobalDeployOverrides: map[string]any{"sp1Verifier": common.Address{0x08}}}
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

// TestDeployImplementationsSelectsStandardSP1Verifier pins which raw SP1 verifier an apply hands
// to DeployImplementations.
func TestDeployImplementationsSelectsStandardSP1Verifier(t *testing.T) {
	const standardVerifier = "0xc3c6dDDAc8829b233Dc6536Ec024775a57b0AF2A"

	// Matches any line reporting a raw verifier, so a reintroduced premature log is caught too.
	verifierLogged := testlog.NewMessageContainsFilter("SP1 verifier")

	deploy := func(t *testing.T, env Env, intent *state.Intent, st *state.State) (*capturingDeployImplementations, *testlog.CapturingHandler, error) {
		capture := new(capturingDeployImplementations)
		lgr, logs := testlog.CaptureLogger(t, slog.LevelDebug)
		env.Logger = lgr
		env.Scripts = &opcm.Scripts{DeployImplementations: capture}
		return capture, logs, DeployImplementations(&env, intent, st)
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
			intent := &state.Intent{L1ChainID: tt.l1ChainID, GlobalDeployOverrides: map[string]any{}}
			capture, logs, err := deploy(t, Env{}, intent, st)
			require.NoError(t, err)
			require.Equal(t, common.HexToAddress(standardVerifier), capture.input.SP1Verifier)
			require.NotNil(t, st.SP1Verifier)
			require.Equal(t, common.HexToAddress(standardVerifier), *st.SP1Verifier)
			require.NotContains(t, intent.GlobalDeployOverrides, "sp1Verifier")
			require.NotNil(t, logs.FindLog(
				verifierLogged,
				testlog.NewAttributesFilter("address", common.HexToAddress(standardVerifier).String()),
			), "should log the verifier it actually deploys against")
		})
	}

	t.Run("explicit override wins", func(t *testing.T) {
		override := common.Address{0x05}
		intent := &state.Intent{L1ChainID: 11155111, GlobalDeployOverrides: map[string]any{"sp1Verifier": override}}
		capture, _, err := deploy(t, Env{}, intent, newSP1VerifierTestState())
		require.NoError(t, err)
		require.Equal(t, override, capture.input.SP1Verifier)
	})

	t.Run("unsupported network passes a zero verifier", func(t *testing.T) {
		st := newSP1VerifierTestState()
		capture, logs, err := deploy(t, Env{}, &state.Intent{L1ChainID: 900}, st)
		require.NoError(t, err)
		require.True(t, capture.ran)
		require.Equal(t, common.Address{}, capture.input.SP1Verifier)
		require.Nil(t, st.SP1Verifier)
		require.NotNil(t, logs.FindLog(testlog.NewLevelFilter(slog.LevelWarn), verifierLogged),
			"should warn that DeployImplementations deploys MockSP1Verifier")
	})

	t.Run("genesis without a verifier rejects chains with a release verifier", func(t *testing.T) {
		capture, _, err := deploy(t, Env{IsGenesis: true}, &state.Intent{L1ChainID: 11155111}, newSP1VerifierTestState())
		require.ErrorContains(t, err, "must set globalDeployOverrides.sp1Verifier")
		require.False(t, capture.ran)
	})

	t.Run("genesis without a verifier passes zero on other chains", func(t *testing.T) {
		st := newSP1VerifierTestState()
		capture, _, err := deploy(t, Env{IsGenesis: true}, &state.Intent{L1ChainID: 900}, st)
		require.NoError(t, err)
		require.True(t, capture.ran)
		require.Equal(t, common.Address{}, capture.input.SP1Verifier)
		require.Nil(t, st.SP1Verifier)
	})

	t.Run("genesis override wins on chains with a release verifier", func(t *testing.T) {
		override := common.Address{0x05}
		intent := &state.Intent{L1ChainID: 11155111, GlobalDeployOverrides: map[string]any{"sp1Verifier": override}}
		capture, _, err := deploy(t, Env{IsGenesis: true}, intent, newSP1VerifierTestState())
		require.NoError(t, err)
		require.Equal(t, override, capture.input.SP1Verifier)
	})

	t.Run("predeployed OPCM records no verifier", func(t *testing.T) {
		opcmAddr := common.Address{0x07}
		st := newSP1VerifierTestState()
		st.ImplementationsDeployment = &addresses.ImplementationsContracts{OpcmV2Impl: opcmAddr}
		intent := &state.Intent{L1ChainID: 11155111, OPCMAddress: &opcmAddr}

		capture, logs, err := deploy(t, Env{}, intent, st)
		require.NoError(t, err)
		require.False(t, capture.ran)
		require.Nil(t, st.SP1Verifier)
		require.Nil(t, logs.FindLog(verifierLogged))
	})

	t.Run("resumed deployment keeps the verifier recorded in state", func(t *testing.T) {
		recorded := common.Address{0x06}
		st := newSP1VerifierTestState()
		st.ImplementationsDeployment = &addresses.ImplementationsContracts{SP1PlonkAdapterImpl: common.Address{0xad}}
		st.SP1Verifier = &recorded

		capture, logs, err := deploy(t, Env{}, &state.Intent{L1ChainID: 11155111}, st)
		require.NoError(t, err)
		require.False(t, capture.ran)
		require.Equal(t, recorded, *st.SP1Verifier)
		require.Nil(t, logs.FindLog(verifierLogged), "resume must not report a verifier it is not using")
	})

	t.Run("resumed mock-backed deployment succeeds", func(t *testing.T) {
		st := newSP1VerifierTestState()
		st.ImplementationsDeployment = &addresses.ImplementationsContracts{SP1PlonkAdapterImpl: common.Address{0xad}}

		capture, _, err := deploy(t, Env{}, &state.Intent{L1ChainID: 900}, st)
		require.NoError(t, err)
		require.False(t, capture.ran)
	})
}
