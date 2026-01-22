package pipeline

import (
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/addresses"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
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

func TestDeploySuperchain_DeploysWithOPCMv1(t *testing.T) {
	lgr := testlog.Logger(t, log.LevelInfo)

	expectedOutput := opcm.DeploySuperchainOutput{
		ProtocolVersionsImpl:  common.BigToAddress(big.NewInt(10)),
		ProtocolVersionsProxy: common.BigToAddress(big.NewInt(11)),
		SuperchainConfigImpl:  common.BigToAddress(big.NewInt(12)),
		SuperchainConfigProxy: common.BigToAddress(big.NewInt(13)),
		SuperchainProxyAdmin:  common.BigToAddress(big.NewInt(14)),
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

	st := &state.State{}

	err := DeploySuperchain(env, intent, st)
	require.NoError(t, err)

	// Verify script was called
	require.Equal(t, 1, mockScript.callCount)

	// Verify input parameters for OPCM v1 (default)
	require.False(t, mockScript.lastInput.IsOPCMv2)
	require.Equal(t, intent.SuperchainRoles.SuperchainGuardian, mockScript.lastInput.Guardian)
	require.Equal(t, intent.SuperchainRoles.ProtocolVersionsOwner, mockScript.lastInput.ProtocolVersionsOwner)
	require.Equal(t, intent.SuperchainRoles.SuperchainProxyAdminOwner, mockScript.lastInput.SuperchainProxyAdminOwner)
	require.Equal(t, rollup.OPStackSupport, mockScript.lastInput.RequiredProtocolVersion)
	require.Equal(t, rollup.OPStackSupport, mockScript.lastInput.RecommendedProtocolVersion)
	require.False(t, mockScript.lastInput.Paused)

	// Verify state was updated with output
	require.NotNil(t, st.SuperchainDeployment)
	require.Equal(t, expectedOutput.SuperchainProxyAdmin, st.SuperchainDeployment.SuperchainProxyAdminImpl)
	require.Equal(t, expectedOutput.SuperchainConfigProxy, st.SuperchainDeployment.SuperchainConfigProxy)
	require.Equal(t, expectedOutput.SuperchainConfigImpl, st.SuperchainDeployment.SuperchainConfigImpl)
	require.Equal(t, expectedOutput.ProtocolVersionsProxy, st.SuperchainDeployment.ProtocolVersionsProxy)
	require.Equal(t, expectedOutput.ProtocolVersionsImpl, st.SuperchainDeployment.ProtocolVersionsImpl)
	require.Equal(t, intent.SuperchainRoles, st.SuperchainRoles)
}

func TestDeploySuperchain_DeploysWithOPCMv2(t *testing.T) {
	lgr := testlog.Logger(t, log.LevelInfo)

	expectedOutput := opcm.DeploySuperchainOutput{
		ProtocolVersionsImpl:  common.Address{}, // Should be zero for OPCM v2
		ProtocolVersionsProxy: common.Address{}, // Should be zero for OPCM v2
		SuperchainConfigImpl:  common.BigToAddress(big.NewInt(12)),
		SuperchainConfigProxy: common.BigToAddress(big.NewInt(13)),
		SuperchainProxyAdmin:  common.BigToAddress(big.NewInt(14)),
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

	opcmV2Flag := common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000010000")
	intent := &state.Intent{
		SuperchainRoles: &addresses.SuperchainRoles{
			SuperchainGuardian:        common.BigToAddress(big.NewInt(1)),
			ProtocolVersionsOwner:     common.BigToAddress(big.NewInt(2)),
			SuperchainProxyAdminOwner: common.BigToAddress(big.NewInt(3)),
		},
		GlobalDeployOverrides: map[string]any{
			"devFeatureBitmap": opcmV2Flag,
		},
	}

	st := &state.State{}

	err := DeploySuperchain(env, intent, st)
	require.NoError(t, err)

	// Verify script was called
	require.Equal(t, 1, mockScript.callCount)

	// Verify input parameters for OPCM v2
	require.True(t, mockScript.lastInput.IsOPCMv2)
	require.Equal(t, intent.SuperchainRoles.SuperchainGuardian, mockScript.lastInput.Guardian)
	require.Equal(t, common.Address{}, mockScript.lastInput.ProtocolVersionsOwner, "ProtocolVersionsOwner should be zero for OPCM v2")
	require.Equal(t, intent.SuperchainRoles.SuperchainProxyAdminOwner, mockScript.lastInput.SuperchainProxyAdminOwner)
	require.Equal(t, params.ProtocolVersion{}, mockScript.lastInput.RequiredProtocolVersion, "RequiredProtocolVersion should be zero for OPCM v2")
	require.Equal(t, params.ProtocolVersion{}, mockScript.lastInput.RecommendedProtocolVersion, "RecommendedProtocolVersion should be zero for OPCM v2")
	require.False(t, mockScript.lastInput.Paused)

	// Verify state was updated with output
	require.NotNil(t, st.SuperchainDeployment)
	require.Equal(t, expectedOutput.SuperchainProxyAdmin, st.SuperchainDeployment.SuperchainProxyAdminImpl)
	require.Equal(t, expectedOutput.SuperchainConfigProxy, st.SuperchainDeployment.SuperchainConfigProxy)
	require.Equal(t, expectedOutput.SuperchainConfigImpl, st.SuperchainDeployment.SuperchainConfigImpl)
	require.Equal(t, common.Address{}, st.SuperchainDeployment.ProtocolVersionsProxy, "ProtocolVersionsProxy should be zero for OPCM v2")
	require.Equal(t, common.Address{}, st.SuperchainDeployment.ProtocolVersionsImpl, "ProtocolVersionsImpl should be zero for OPCM v2")
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
