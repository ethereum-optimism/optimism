package opcm

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func compareBigInt(a, b *big.Int) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return a.Cmp(b) == 0
}

// compareAddGameTypeInputs compares two AddGameTypeInput structs with special handling for *big.Int fields. We can't
// use require.Equal directly because zero *big.Int structs can either be nil or a zero value, which trips up the
// equality checker.
func compareAddGameTypeInputs(t *testing.T, expected, actual AddGameTypeInput) {
	require.Equal(t, expected.L1ProxyAdminOwner, actual.L1ProxyAdminOwner)
	require.Equal(t, expected.OPCMImpl, actual.OPCMImpl)
	require.Equal(t, expected.SystemConfigProxy, actual.SystemConfigProxy)
	require.Equal(t, expected.OPChainProxyAdmin, actual.OPChainProxyAdmin)
	require.Equal(t, expected.DelayedWETHProxy, actual.DelayedWETHProxy)
	require.Equal(t, expected.DisputeGameType, actual.DisputeGameType)
	require.Equal(t, expected.DisputeAbsolutePrestate, actual.DisputeAbsolutePrestate)
	require.Equal(t, expected.DisputeClockExtension, actual.DisputeClockExtension)
	require.Equal(t, expected.DisputeMaxClockDuration, actual.DisputeMaxClockDuration)
	require.Equal(t, expected.VM, actual.VM)
	require.Equal(t, expected.Permissioned, actual.Permissioned)
	require.Equal(t, expected.SaltMixer, actual.SaltMixer)

	// Special handling for *big.Int fields
	require.True(t, compareBigInt(expected.DisputeMaxGameDepth, actual.DisputeMaxGameDepth))
	require.True(t, compareBigInt(expected.DisputeSplitDepth, actual.DisputeSplitDepth))
	require.True(t, compareBigInt(expected.InitialBond, actual.InitialBond))
}

func TestAddGameTypeInput_MarshalUnmarshalJSON(t *testing.T) {
	// Define test cases
	testCases := []struct {
		name  string
		input AddGameTypeInput
	}{
		{
			name: "basic",
			input: AddGameTypeInput{
				L1ProxyAdminOwner:       common.HexToAddress("0x1111111111111111111111111111111111111111"),
				OPCMImpl:                common.HexToAddress("0x2222222222222222222222222222222222222222"),
				SystemConfigProxy:       common.HexToAddress("0x3333333333333333333333333333333333333333"),
				OPChainProxyAdmin:       common.HexToAddress("0x4444444444444444444444444444444444444444"),
				DelayedWETHProxy:        common.HexToAddress("0x5555555555555555555555555555555555555555"),
				DisputeGameType:         1,
				DisputeAbsolutePrestate: common.HexToHash("0x6666666666666666666666666666666666666666666666666666666666666666"),
				DisputeMaxGameDepth:     big.NewInt(100),
				DisputeSplitDepth:       big.NewInt(10),
				DisputeClockExtension:   1000,
				DisputeMaxClockDuration: 2000,
				InitialBond:             big.NewInt(5000000000000000000), // 5 ETH
				VM:                      common.HexToAddress("0x7777777777777777777777777777777777777777"),
				Permissioned:            true,
				SaltMixer:               "salt_mixer_value",
			},
		},
		{
			name: "nil big.Int fields",
			input: AddGameTypeInput{
				L1ProxyAdminOwner:       common.HexToAddress("0x1111111111111111111111111111111111111111"),
				OPCMImpl:                common.HexToAddress("0x2222222222222222222222222222222222222222"),
				SystemConfigProxy:       common.HexToAddress("0x3333333333333333333333333333333333333333"),
				OPChainProxyAdmin:       common.HexToAddress("0x4444444444444444444444444444444444444444"),
				DelayedWETHProxy:        common.HexToAddress("0x5555555555555555555555555555555555555555"),
				DisputeGameType:         1,
				DisputeAbsolutePrestate: common.HexToHash("0x6666666666666666666666666666666666666666666666666666666666666666"),
				DisputeMaxGameDepth:     nil, // nil big.Int
				DisputeSplitDepth:       nil, // nil big.Int
				DisputeClockExtension:   1000,
				DisputeMaxClockDuration: 2000,
				InitialBond:             nil, // nil big.Int
				VM:                      common.HexToAddress("0x7777777777777777777777777777777777777777"),
				Permissioned:            false,
				SaltMixer:               "salt_mixer_value",
			},
		},
	}

	// Run test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			data, err := json.Marshal(tc.input)
			require.NoError(t, err)

			var unmarshaled AddGameTypeInput
			err = json.Unmarshal(data, &unmarshaled)
			require.NoError(t, err)

			compareAddGameTypeInputs(t, tc.input, unmarshaled)

			newData, err := json.Marshal(unmarshaled)
			require.NoError(t, err)
			require.JSONEq(t, string(data), string(newData))
		})
	}
}
