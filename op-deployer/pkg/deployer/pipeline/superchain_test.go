package pipeline

import (
	"context"
	"fmt"
	"log/slog"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/addresses"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/artifacts"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/broadcaster"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/forge"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/testutil"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/env"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils/devnet"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
)

// mockDeploySuperchainScript captures the input and returns a mock output
type mockDeploySuperchainScript struct {
	lastInput   opcm.DeploySuperchainInput
	outputToRet opcm.DeploySuperchainOutput
	errorToRet  error
	callCount   int
}

func (m *mockDeploySuperchainScript) Run(input opcm.DeploySuperchainInput) (opcm.DeploySuperchainOutput, error) {
	m.lastInput = input
	m.callCount++
	return m.outputToRet, m.errorToRet
}

func (m *mockDeploySuperchainScript) ABI() abi.ABI {
	return abi.ABI{}
}

func (m *mockDeploySuperchainScript) Name() string {
	return "MockDeploySuperchain"
}

func (m *mockDeploySuperchainScript) Call(input []byte) (result []byte, err error) {
	return nil, nil
}

func TestDeploySuperchain_SkipsWhenAlreadyDeployed(t *testing.T) {
	lgr := testlog.Logger(t, log.LevelInfo)
	mockScript := &mockDeploySuperchainScript{}

	env := &Env{
		Logger: lgr,
		Scripts: &opcm.Scripts{
			DeploySuperchain: mockScript,
		},
	}

	intent := &state.Intent{
		SuperchainRoles: &addresses.SuperchainRoles{
			SuperchainGuardian:        common.BigToAddress(big.NewInt(1)),
			ProtocolVersionsOwner:     common.BigToAddress(big.NewInt(2)),
			SuperchainProxyAdminOwner: common.BigToAddress(big.NewInt(3)),
		},
	}

	st := &state.State{
		SuperchainDeployment: &addresses.SuperchainContracts{
			SuperchainConfigProxy: common.BigToAddress(big.NewInt(100)),
		},
	}

	err := DeploySuperchain(env, intent, st)
	require.NoError(t, err)
	require.Equal(t, 0, mockScript.callCount, "script should not be called when superchain already deployed")
}

func TestDeploySuperchain_Deploys(t *testing.T) {
	testCases := []struct {
		name            string
		useOPCMv2       bool
		expectProtocolV bool
	}{
		{
			name:            "v1",
			useOPCMv2:       false,
			expectProtocolV: true,
		},
		{
			name:            "v2",
			useOPCMv2:       true,
			expectProtocolV: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			lgr := testlog.Logger(t, log.LevelInfo)

			expectedOutput := opcm.DeploySuperchainOutput{
				ProtocolVersionsImpl:  common.Address{},
				ProtocolVersionsProxy: common.Address{},
				SuperchainConfigImpl:  common.BigToAddress(big.NewInt(12)),
				SuperchainConfigProxy: common.BigToAddress(big.NewInt(13)),
				SuperchainProxyAdmin:  common.BigToAddress(big.NewInt(14)),
			}

			if tc.expectProtocolV {
				expectedOutput.ProtocolVersionsImpl = common.BigToAddress(big.NewInt(10))
				expectedOutput.ProtocolVersionsProxy = common.BigToAddress(big.NewInt(11))
			}

			mockScript := &mockDeploySuperchainScript{
				outputToRet: expectedOutput,
			}

			env := &Env{
				Logger: lgr,
				Scripts: &opcm.Scripts{
					DeploySuperchain: mockScript,
				},
			}

			intent := &state.Intent{
				SuperchainRoles: &addresses.SuperchainRoles{
					SuperchainGuardian:        common.BigToAddress(big.NewInt(1)),
					ProtocolVersionsOwner:     common.BigToAddress(big.NewInt(2)),
					SuperchainProxyAdminOwner: common.BigToAddress(big.NewInt(3)),
				},
				GlobalDeployOverrides: map[string]any{},
			}

			if tc.useOPCMv2 {
				opcmV2Flag := common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000010000")
				intent.GlobalDeployOverrides["devFeatureBitmap"] = opcmV2Flag
			}

			st := &state.State{}

			err := DeploySuperchain(env, intent, st)
			require.NoError(t, err)

			// Verify script was called
			require.Equal(t, 1, mockScript.callCount)

			// Verify input parameters
			require.Equal(t, tc.useOPCMv2, mockScript.lastInput.IsOPCMv2)
			require.Equal(t, intent.SuperchainRoles.SuperchainGuardian, mockScript.lastInput.Guardian)
			require.Equal(t, intent.SuperchainRoles.SuperchainProxyAdminOwner, mockScript.lastInput.SuperchainProxyAdminOwner)
			require.False(t, mockScript.lastInput.Paused)

			if tc.useOPCMv2 {
				require.Equal(t, common.Address{}, mockScript.lastInput.ProtocolVersionsOwner, "ProtocolVersionsOwner should be zero for OPCM v2")
				require.Equal(t, params.ProtocolVersion{}, mockScript.lastInput.RequiredProtocolVersion, "RequiredProtocolVersion should be zero for OPCM v2")
				require.Equal(t, params.ProtocolVersion{}, mockScript.lastInput.RecommendedProtocolVersion, "RecommendedProtocolVersion should be zero for OPCM v2")
			} else {
				require.Equal(t, intent.SuperchainRoles.ProtocolVersionsOwner, mockScript.lastInput.ProtocolVersionsOwner)
				require.Equal(t, rollup.OPStackSupport, mockScript.lastInput.RequiredProtocolVersion)
				require.Equal(t, rollup.OPStackSupport, mockScript.lastInput.RecommendedProtocolVersion)
			}

			// Verify state was updated with output
			require.NotNil(t, st.SuperchainDeployment)
			require.Equal(t, expectedOutput.SuperchainProxyAdmin, st.SuperchainDeployment.SuperchainProxyAdminImpl)
			require.Equal(t, expectedOutput.SuperchainConfigProxy, st.SuperchainDeployment.SuperchainConfigProxy)
			require.Equal(t, expectedOutput.SuperchainConfigImpl, st.SuperchainDeployment.SuperchainConfigImpl)

			if tc.expectProtocolV {
				require.Equal(t, expectedOutput.ProtocolVersionsProxy, st.SuperchainDeployment.ProtocolVersionsProxy)
				require.Equal(t, expectedOutput.ProtocolVersionsImpl, st.SuperchainDeployment.ProtocolVersionsImpl)
				require.Equal(t, intent.SuperchainRoles, st.SuperchainRoles)
			} else {
				require.Equal(t, common.Address{}, st.SuperchainDeployment.ProtocolVersionsProxy, "ProtocolVersionsProxy should be zero for OPCM v2")
				require.Equal(t, common.Address{}, st.SuperchainDeployment.ProtocolVersionsImpl, "ProtocolVersionsImpl should be zero for OPCM v2")
			}
		})
	}
}

func TestShouldDeploySuperchain(t *testing.T) {
	tests := []struct {
		name     string
		intent   *state.Intent
		st       *state.State
		expected bool
	}{
		{
			name:   "should deploy when superchain deployment is nil",
			intent: &state.Intent{},
			st: &state.State{
				SuperchainDeployment: nil,
			},
			expected: true,
		},
		{
			name:   "should not deploy when superchain deployment exists",
			intent: &state.Intent{},
			st: &state.State{
				SuperchainDeployment: &addresses.SuperchainContracts{
					SuperchainConfigProxy: common.BigToAddress(big.NewInt(1)),
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shouldDeploySuperchain(tt.intent, tt.st)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestDeploySuperchain_WithForge(t *testing.T) {
	testCases := []struct {
		name            string
		useOPCMv2       bool
		expectProtocolV bool
	}{
		{
			name:            "v1",
			useOPCMv2:       false,
			expectProtocolV: true,
		},
		{
			name:            "v2",
			useOPCMv2:       true,
			expectProtocolV: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			tmpDir := t.TempDir()

			// Extract embedded artifacts
			embeddedArtifactsFS, err := artifacts.ExtractEmbedded(tmpDir)
			require.NoError(t, err)

			// Create Forge client
			forgeClient, err := forge.NewStandardClient(fmt.Sprintf("%v", embeddedArtifactsFS))
			require.NoError(t, err)

			// Create a test host for other scripts
			// We use LocalArtifacts which should have compatible versions
			_, afacts := testutil.LocalArtifacts(t)
			lgr := testlog.Logger(t, slog.LevelInfo)
			anvil, err := devnet.NewAnvil(lgr)
			require.NoError(t, err)
			require.NoError(t, anvil.Start())
			t.Cleanup(func() {
				require.NoError(t, anvil.Stop())
			})

			l1RPCUrl := anvil.RPCUrl()

			host, err := env.DefaultScriptHost(
				broadcaster.NoopBroadcaster(),
				lgr,
				common.Address{'D'},
				afacts,
			)
			require.NoError(t, err)

			// Load scripts
			opcmScripts, err := opcm.NewScripts(host)
			require.NoError(t, err)

			// Create test input
			intent := &state.Intent{
				SuperchainRoles: &addresses.SuperchainRoles{
					SuperchainProxyAdminOwner: common.BigToAddress(big.NewInt(1)),
					ProtocolVersionsOwner:     common.BigToAddress(big.NewInt(2)),
					SuperchainGuardian:        common.BigToAddress(big.NewInt(3)),
				},
				GlobalDeployOverrides: map[string]any{},
			}

			if tc.useOPCMv2 {
				opcmV2Flag := common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000010000")
				intent.GlobalDeployOverrides["devFeatureBitmap"] = opcmV2Flag
			}

			st := &state.State{
				Version: 1,
			}

			// Create Env with Forge enabled
			pEnv := &Env{
				Logger:      lgr,
				Scripts:     opcmScripts,
				ForgeClient: forgeClient,
				UseForge:    true,
				Context:     ctx,
				Broadcaster: broadcaster.NoopBroadcaster(),
				StateWriter: NoopStateWriter(),
				PrivateKey:  testutil.AnvilDefaultPrivateKey,
				L1RPCUrl:    l1RPCUrl,
			}

			// Test DeploySuperchain with Forge
			err = DeploySuperchain(pEnv, intent, st)
			require.NoError(t, err)

			// Verify the deployment was successful
			require.NotNil(t, st.SuperchainDeployment)
			require.NotNil(t, st.SuperchainRoles)

			// Verify addresses are set
			require.NotEqual(t, common.Address{}, st.SuperchainDeployment.SuperchainProxyAdminImpl)
			require.NotEqual(t, common.Address{}, st.SuperchainDeployment.SuperchainConfigProxy)
			require.NotEqual(t, common.Address{}, st.SuperchainDeployment.SuperchainConfigImpl)

			if tc.expectProtocolV {
				require.NotEqual(t, common.Address{}, st.SuperchainDeployment.ProtocolVersionsProxy)
				require.NotEqual(t, common.Address{}, st.SuperchainDeployment.ProtocolVersionsImpl)
				// Verify roles match
				require.Equal(t, intent.SuperchainRoles.ProtocolVersionsOwner, st.SuperchainRoles.ProtocolVersionsOwner)
			} else {
				require.Equal(t, common.Address{}, st.SuperchainDeployment.ProtocolVersionsProxy, "ProtocolVersionsProxy should be zero for OPCM v2")
				require.Equal(t, common.Address{}, st.SuperchainDeployment.ProtocolVersionsImpl, "ProtocolVersionsImpl should be zero for OPCM v2")
			}

			// Verify roles match
			require.Equal(t, intent.SuperchainRoles.SuperchainProxyAdminOwner, st.SuperchainRoles.SuperchainProxyAdminOwner)
			require.Equal(t, intent.SuperchainRoles.SuperchainGuardian, st.SuperchainRoles.SuperchainGuardian)
		})
	}
}

func TestDeploySuperchain_WithForgeEverywhere(t *testing.T) {
	testCases := []struct {
		name            string
		useOPCMv2       bool
		expectProtocolV bool
	}{
		{
			name:            "v1",
			useOPCMv2:       false,
			expectProtocolV: true,
		},
		{
			name:            "v2",
			useOPCMv2:       true,
			expectProtocolV: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			tmpDir := t.TempDir()

			// Extract embedded artifacts
			embeddedArtifactsFS, err := artifacts.ExtractEmbedded(tmpDir)
			require.NoError(t, err)

			// Create Forge client
			forgeClient, err := forge.NewStandardClient(fmt.Sprintf("%v", embeddedArtifactsFS))
			require.NoError(t, err)

			// Create a test host for other scripts
			_, afacts := testutil.LocalArtifacts(t)
			lgr := testlog.Logger(t, slog.LevelInfo)
			anvil, err := devnet.NewAnvil(lgr)
			require.NoError(t, err)
			require.NoError(t, anvil.Start())
			t.Cleanup(func() {
				require.NoError(t, anvil.Stop())
			})

			l1RPCUrl := anvil.RPCUrl()

			host, err := env.DefaultScriptHost(
				broadcaster.NoopBroadcaster(),
				lgr,
				common.Address{'D'},
				afacts,
			)
			require.NoError(t, err)

			// Load scripts
			opcmScripts, err := opcm.NewScripts(host)
			require.NoError(t, err)

			// Create test input
			intent := &state.Intent{
				SuperchainRoles: &addresses.SuperchainRoles{
					SuperchainProxyAdminOwner: common.BigToAddress(big.NewInt(1)),
					ProtocolVersionsOwner:     common.BigToAddress(big.NewInt(2)),
					SuperchainGuardian:        common.BigToAddress(big.NewInt(3)),
				},
				GlobalDeployOverrides: map[string]any{},
			}

			if tc.useOPCMv2 {
				opcmV2Flag := common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000010000")
				intent.GlobalDeployOverrides["devFeatureBitmap"] = opcmV2Flag
			}

			st := &state.State{
				Version: 1,
			}

			// Create Env with UseForge enabled
			pEnv := &Env{
				Logger:      lgr,
				Scripts:     opcmScripts,
				ForgeClient: forgeClient,
				UseForge:    true,
				Context:     ctx,
				Broadcaster: broadcaster.NoopBroadcaster(),
				StateWriter: NoopStateWriter(),
				PrivateKey:  testutil.AnvilDefaultPrivateKey,
				L1RPCUrl:    l1RPCUrl,
			}

			// Test DeploySuperchain with Forge
			err = DeploySuperchain(pEnv, intent, st)
			require.NoError(t, err)

			// Verify the deployment was successful
			require.NotNil(t, st.SuperchainDeployment)
			require.NotNil(t, st.SuperchainRoles)

			// Verify addresses are set
			require.NotEqual(t, common.Address{}, st.SuperchainDeployment.SuperchainProxyAdminImpl)
			require.NotEqual(t, common.Address{}, st.SuperchainDeployment.SuperchainConfigProxy)
			require.NotEqual(t, common.Address{}, st.SuperchainDeployment.SuperchainConfigImpl)

			if tc.expectProtocolV {
				require.NotEqual(t, common.Address{}, st.SuperchainDeployment.ProtocolVersionsProxy)
				require.NotEqual(t, common.Address{}, st.SuperchainDeployment.ProtocolVersionsImpl)
			} else {
				require.Equal(t, common.Address{}, st.SuperchainDeployment.ProtocolVersionsProxy, "ProtocolVersionsProxy should be zero for OPCM v2")
				require.Equal(t, common.Address{}, st.SuperchainDeployment.ProtocolVersionsImpl, "ProtocolVersionsImpl should be zero for OPCM v2")
			}
		})
	}
}

func TestDeploySuperchain_WithForge_ManualCall(t *testing.T) {
	testCases := []struct {
		name            string
		useOPCMv2       bool
		expectProtocolV bool
	}{
		{
			name:            "v1",
			useOPCMv2:       false,
			expectProtocolV: true,
		},
		{
			name:            "v2",
			useOPCMv2:       true,
			expectProtocolV: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			tmpDir := t.TempDir()

			// Extract embedded artifacts
			embeddedArtifactsFS, err := artifacts.ExtractEmbedded(tmpDir)
			require.NoError(t, err)

			// Create Forge client
			forgeClient, err := forge.NewStandardClient(fmt.Sprintf("%v", embeddedArtifactsFS))
			require.NoError(t, err)

			// Create Forge caller directly
			deploySuperchain := opcm.NewDeploySuperchainForgeCaller(forgeClient)

			// Create input matching what DeploySuperchain would use
			input := opcm.DeploySuperchainInput{
				ProtocolVersionsOwner:      common.Address{},
				RequiredProtocolVersion:    params.ProtocolVersion{},
				RecommendedProtocolVersion: params.ProtocolVersion{},
				Guardian:                   common.BigToAddress(big.NewInt(1)),
				SuperchainProxyAdminOwner:  common.BigToAddress(big.NewInt(3)),
				Paused:                     false,
				IsOPCMv2:                   tc.useOPCMv2,
			}

			if !tc.useOPCMv2 {
				input.ProtocolVersionsOwner = common.BigToAddress(big.NewInt(2))
				input.RequiredProtocolVersion = params.ProtocolVersion(rollup.OPStackSupport)
				input.RecommendedProtocolVersion = params.ProtocolVersion(rollup.OPStackSupport)
			}

			// Call Forge script
			output, recompiled, err := deploySuperchain(ctx, input)
			require.NoError(t, err)
			require.False(t, recompiled, "script should not be recompiled")
			require.NotNil(t, output)

			// Verify output addresses are set
			require.NotEqual(t, common.Address{}, output.SuperchainProxyAdmin)
			require.NotEqual(t, common.Address{}, output.SuperchainConfigProxy)
			require.NotEqual(t, common.Address{}, output.SuperchainConfigImpl)

			if tc.expectProtocolV {
				require.NotEqual(t, common.Address{}, output.ProtocolVersionsProxy)
				require.NotEqual(t, common.Address{}, output.ProtocolVersionsImpl)
			} else {
				require.Equal(t, common.Address{}, output.ProtocolVersionsProxy, "ProtocolVersionsProxy should be zero for OPCM v2")
				require.Equal(t, common.Address{}, output.ProtocolVersionsImpl, "ProtocolVersionsImpl should be zero for OPCM v2")
			}
		})
	}
}
