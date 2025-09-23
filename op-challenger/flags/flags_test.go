package flags

import (
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-challenger/config"
	opservice "github.com/ethereum-optimism/optimism/op-service"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/superchain"

	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

// TestUniqueFlags asserts that all flag names are unique, to avoid accidental conflicts between the many flags.
func TestUniqueFlags(t *testing.T) {
	seenCLI := make(map[string]struct{})
	for _, flag := range Flags {
		for _, name := range flag.Names() {
			if _, ok := seenCLI[name]; ok {
				t.Errorf("duplicate flag %s", name)
				continue
			}
			seenCLI[name] = struct{}{}
		}
	}
}

// TestUniqueEnvVars asserts that all flag env vars are unique, to avoid accidental conflicts between the many flags.
func TestUniqueEnvVars(t *testing.T) {
	seenCLI := make(map[string]struct{})
	for _, flag := range Flags {
		envVar := envVarForFlag(flag)
		if _, ok := seenCLI[envVar]; envVar != "" && ok {
			t.Errorf("duplicate flag env var %s", envVar)
			continue
		}
		seenCLI[envVar] = struct{}{}
	}
}

func TestCorrectEnvVarPrefix(t *testing.T) {
	for _, flag := range Flags {
		envVar := envVarForFlag(flag)
		if envVar == "" {
			t.Errorf("Failed to find EnvVar for flag %v", flag.Names()[0])
		}
		if !strings.HasPrefix(envVar, "OP_CHALLENGER_") {
			t.Errorf("Flag %v env var (%v) does not start with OP_CHALLENGER_", flag.Names()[0], envVar)
		}
		if strings.Contains(envVar, "__") {
			t.Errorf("Flag %v env var (%v) has duplicate underscores", flag.Names()[0], envVar)
		}
	}
}

func envVarForFlag(flag cli.Flag) string {
	values := reflect.ValueOf(flag)
	envVarValue := values.Elem().FieldByName("EnvVars")
	if envVarValue == (reflect.Value{}) || envVarValue.Len() == 0 {
		return ""
	}
	return envVarValue.Index(0).String()
}

func TestEnvVarFormat(t *testing.T) {
	for _, flag := range Flags {
		flag := flag
		flagName := flag.Names()[0]

		skippedFlags := []string{
			txmgr.FeeLimitMultiplierFlagName,
			txmgr.TxSendTimeoutFlagName,
			txmgr.TxNotInMempoolTimeoutFlagName,
		}

		t.Run(flagName, func(t *testing.T) {
			if slices.Contains(skippedFlags, flagName) {
				t.Skipf("Skipping flag %v which is known to not have a standard flag name <-> env var conversion", flagName)
			}
			envFlagGetter, ok := flag.(interface {
				GetEnvVars() []string
			})
			envFlags := envFlagGetter.GetEnvVars()
			require.True(t, ok, "must be able to cast the flag to an EnvVar interface")
			require.Equal(t, 1, len(envFlags), "flags should have exactly one env var")
			expectedEnvVar := opservice.FlagNameToEnvVarName(flagName, "OP_CHALLENGER")
			require.Equal(t, expectedEnvVar, envFlags[0])
		})
	}
}

func TestResponseDelayFlag(t *testing.T) {
	t.Run("IncludedInOptionalFlags", func(t *testing.T) {
		require.Contains(t, optionalFlags, ResponseDelayFlag, "ResponseDelayFlag should be in optionalFlags")
	})

	t.Run("HasCorrectEnvVar", func(t *testing.T) {
		envVars := ResponseDelayFlag.GetEnvVars()
		require.Len(t, envVars, 1, "ResponseDelayFlag should have exactly one env var")
		require.Equal(t, "OP_CHALLENGER_RESPONSE_DELAY", envVars[0], "ResponseDelayFlag should have correct env var")
	})

	t.Run("DefaultValue", func(t *testing.T) {
		require.Equal(t, time.Duration(config.DefaultResponseDelay), ResponseDelayFlag.Value, "ResponseDelayFlag should have correct default value")
	})
}

func TestResponseDelayAfterFlag(t *testing.T) {
	t.Run("IncludedInOptionalFlags", func(t *testing.T) {
		require.Contains(t, optionalFlags, ResponseDelayAfterFlag, "ResponseDelayAfterFlag should be in optionalFlags")
	})

	t.Run("HasCorrectEnvVar", func(t *testing.T) {
		envVars := ResponseDelayAfterFlag.GetEnvVars()
		require.Len(t, envVars, 1, "ResponseDelayAfterFlag should have exactly one env var")
		require.Equal(t, "OP_CHALLENGER_RESPONSE_DELAY_AFTER", envVars[0], "ResponseDelayAfterFlag should have correct env var")
	})

	t.Run("DefaultValue", func(t *testing.T) {
		require.Equal(t, uint64(config.DefaultResponseDelayAfter), ResponseDelayAfterFlag.Value, "ResponseDelayAfterFlag should have correct default value")
	})
}

func TestMultipleNetworksMustShareDisputeGameFactory(t *testing.T) {
	addrs := map[string]superchain.AddressesConfig{
		"a1": {DisputeGameFactoryProxy: &common.Address{0xAA}},
		"a2": {DisputeGameFactoryProxy: &common.Address{0xAA}},
		"b1": {DisputeGameFactoryProxy: &common.Address{0xBB}},
	}
	source := func(network string) (superchain.AddressesConfig, error) {
		return addrs[network], nil
	}

	t.Run("SameAddress", func(t *testing.T) {
		actual, err := FactoryAddressForNetworks([]string{"a1", "a2"}, source)
		require.NoError(t, err)
		require.Equal(t, actual, common.Address{0xAA})
	})

	t.Run("DifferentAddresses", func(t *testing.T) {
		actual, err := FactoryAddressForNetworks([]string{"a1", "a2", "b1"}, source)
		require.ErrorContains(t, err, "different dispute game factories")
		require.Equal(t, actual, common.Address{})
	})

	t.Run("SingleNetwork", func(t *testing.T) {
		actual, err := FactoryAddressForNetworks([]string{"b1"}, source)
		require.NoError(t, err)
		require.Equal(t, actual, common.Address{0xBB})
	})
}
