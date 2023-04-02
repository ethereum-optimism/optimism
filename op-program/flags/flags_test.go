package flags

import (
	"reflect"
	"strings"
	"testing"
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

func TestCorrectEnvVarPrefix(t *testing.T) {
	for _, flag := range Flags {
		values := reflect.ValueOf(flag)
		envVarValue := values.FieldByName("EnvVar")
		if envVarValue == (reflect.Value{}) {
			t.Errorf("Failed to find EnvVar for flag %v", flag.GetName())
			continue
		}
		envVar := envVarValue.String()
		if envVar[:len("OP_PROGRAM_")] != "OP_PROGRAM_" {
			t.Errorf("Flag %v env var (%v) does not start with OP_PROGRAM_", flag.GetName(), envVar)
		}
		if strings.Contains(envVar, "__") {
			t.Errorf("Flag %v env var (%v) has duplicate underscores", flag.GetName(), envVar)
		}
	}
}
