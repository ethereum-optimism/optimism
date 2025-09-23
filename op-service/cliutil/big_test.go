package cliutil

import (
	"flag"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

func TestBigIntFlag(t *testing.T) {
	validBigIntStr := "123456789012345678901234567890"
	expectedBigInt, _ := new(big.Int).SetString(validBigIntStr, 10)

	tests := []struct {
		name        string
		flagValue   string
		expectedVal *big.Int
		expectErr   bool
	}{
		{
			name:        "valid big int",
			flagValue:   validBigIntStr,
			expectedVal: expectedBigInt,
			expectErr:   false,
		},
		{
			name:        "valid hex big int",
			flagValue:   "0x1234",
			expectedVal: big.NewInt(0x1234),
			expectErr:   false,
		},
		{
			name:        "invalid hex big int",
			flagValue:   "0xgibberish",
			expectedVal: nil,
			expectErr:   true,
		},
		{
			name:        "empty hex big int",
			flagValue:   "0x",
			expectedVal: nil,
			expectErr:   true,
		},
		{
			name:        "invalid big int string",
			flagValue:   "not-a-number",
			expectedVal: nil,
			expectErr:   true,
		},
		{
			name:        "empty string value for flag",
			flagValue:   "",
			expectedVal: nil,
			expectErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := flag.NewFlagSet("test", flag.ContinueOnError)
			flagName := "foo"
			fs.String(flagName, tt.flagValue, "doc")
			val, err := BigIntFlag(cli.NewContext(nil, fs, nil), flagName)

			if tt.expectErr {
				require.Nil(t, val)
				require.Error(t, err)
			} else {
				require.NotNil(t, val)
				require.Equal(t, 0, tt.expectedVal.Cmp(val), "expected %s, got %s", tt.expectedVal.String(), val.String())
			}
		})
	}
}
