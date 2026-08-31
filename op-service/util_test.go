package op_service

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"

	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

func TestCLIFlagsToEnvVars(t *testing.T) {
	flags := []cli.Flag{
		&cli.StringFlag{
			Name:    "test",
			EnvVars: []string{"OP_NODE_TEST_VAR"},
		},
		&cli.IntFlag{
			Name: "no env var",
		},
	}
	res := cliFlagsToEnvVars(flags)
	require.Contains(t, res, "OP_NODE_TEST_VAR")
}

func TestValidateEnvVars(t *testing.T) {
	provided := []string{"OP_BATCHER_CONFIG=true", "OP_BATCHER_FAKE=false", "LD_PRELOAD=/lib/fake.so"}
	defined := map[string]struct{}{
		"OP_BATCHER_CONFIG": {},
		"OP_BATCHER_OTHER":  {},
	}
	invalids := validateEnvVars("OP_BATCHER", provided, defined)
	require.ElementsMatch(t, invalids, []string{"OP_BATCHER_FAKE=false"})
}

func TestEnvVarsWithPrefixes(t *testing.T) {
	provided := []string{
		"OP_NODE_ALTDA_ENABLED=true",
		"OP_BATCHER_ALTDA_DA_SERVER=https://example.com",
		"OP_NODE_L1_ETH_RPC=https://example.com",
		"OP_NODE_ALTDA_VERIFY_ON_READ=",
	}

	require.Equal(t, []string{
		"OP_BATCHER_ALTDA_DA_SERVER",
		"OP_NODE_ALTDA_ENABLED",
		"OP_NODE_ALTDA_VERIFY_ON_READ",
	}, envVarsWithPrefixes(provided, "OP_NODE_ALTDA_", "OP_BATCHER_ALTDA_"))
	require.Empty(t, envVarsWithPrefixes(provided, ""))
}

func TestRejectEnvVarPrefixes(t *testing.T) {
	t.Setenv("OPTIMISM_TEST_REMOVED_ALTDA_ENABLED", "false")

	err := RejectEnvVarPrefixes("OPTIMISM_TEST_REMOVED_ALTDA_")
	require.EqualError(t, err, "unsupported environment variables are set: OPTIMISM_TEST_REMOVED_ALTDA_ENABLED")
}

func TestParse256BitChainID(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected common.Hash
		err      bool
	}{
		{
			name:     "valid int",
			input:    "12345",
			expected: common.Hash{30: 0x30, 31: 0x39},
			err:      false,
		},
		{
			name:     "invalid hash",
			input:    common.Hash{0x00: 0xff}.String(),
			expected: common.Hash{0x00: 0xff},
			err:      false,
		},
		{
			name:  "hash overflow",
			input: "0xff0000000000000000000000000000000000000000000000000000000000000000",
			err:   true,
		},
		{
			name: "number overflow",
			// (2^256 - 1) + 1
			input: "115792089237316195423570985008687907853269984665640564039457584007913129639936",
			err:   true,
		},
		{
			name:  "invalid hex",
			input: "0xnope",
			err:   true,
		},
		{
			name:  "invalid number",
			input: "nope",
			err:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := Parse256BitChainID(tt.input)
			if tt.err {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expected, res)
			}
		})
	}
}
