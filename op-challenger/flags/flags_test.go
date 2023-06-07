package flags

import (
	"reflect"
	"strings"
	"testing"

	"github.com/urfave/cli"
)

// TestUniqueFlags asserts that all flag names are unique, to avoid accidental conflicts between the many flags.
func TestUniqueFlags(t *testing.T) {
	seenCLI := make(map[string]struct{})
	for _, flag := range Flags {
		name := flag.GetName()
		if _, ok := seenCLI[name]; ok {
			t.Errorf("duplicate flag %s", name)
			continue
		}
		seenCLI[name] = struct{}{}
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
			t.Errorf("Failed to find EnvVar for flag %v", flag.GetName())
		}
		if !strings.HasPrefix(envVar, "OP_CHALLENGER_") {
			t.Errorf("Flag %v env var (%v) does not start with OP_CHALLENGER_", flag.GetName(), envVar)
		}
		if strings.Contains(envVar, "__") {
			t.Errorf("Flag %v env var (%v) has duplicate underscores", flag.GetName(), envVar)
		}
	}
}

func envVarForFlag(flag cli.Flag) string {
	values := reflect.ValueOf(flag)
	envVarValue := values.FieldByName("EnvVar")
	if envVarValue == (reflect.Value{}) {
		return ""
	}
	return envVarValue.String()
}
