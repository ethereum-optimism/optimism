package flags

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	opservice "github.com/ethereum-optimism/optimism/op-service"

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
		if !strings.HasPrefix(envVar, fmt.Sprintf("%s_", envVarPrefix)) {
			t.Errorf("Flag %v env var (%v) does not start with %s_", flag.Names()[0], envVar, envVarPrefix)
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

		t.Run(flagName, func(t *testing.T) {
			envFlagGetter, ok := flag.(interface {
				GetEnvVars() []string
			})
			envFlags := envFlagGetter.GetEnvVars()
			require.True(t, ok, "must be able to cast the flag to an EnvVar interface")
			require.Equal(t, 1, len(envFlags), "flags should have exactly one env var")
			expectedEnvVar := opservice.FlagNameToEnvVarName(flagName, envVarPrefix)
			require.Equal(t, expectedEnvVar, envFlags[0])
		})
	}
}
