package flags

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

func TestFlags(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		envVars  map[string]string
		expected struct {
			fixTraefik      bool
			conductorConfig string
			environment     string
		}
	}{
		{
			name: "default values",
			args: []string{"inspect", "test-enclave"},
			expected: struct {
				fixTraefik      bool
				conductorConfig string
				environment     string
			}{
				fixTraefik:      false,
				conductorConfig: "",
				environment:     "",
			},
		},
		{
			name: "cli flags set",
			args: []string{
				"inspect",
				"--fix-traefik",
				"--conductor-config-path", "/tmp/conductor.toml",
				"--environment-path", "/tmp/env.json",
				"test-enclave",
			},
			expected: struct {
				fixTraefik      bool
				conductorConfig string
				environment     string
			}{
				fixTraefik:      true,
				conductorConfig: "/tmp/conductor.toml",
				environment:     "/tmp/env.json",
			},
		},
		{
			name: "environment variables",
			args: []string{"inspect", "test-enclave"},
			envVars: map[string]string{
				"KURTOSIS_INSPECT_FIX_TRAEFIK":      "true",
				"KURTOSIS_INSPECT_CONDUCTOR_CONFIG": "/env/conductor.toml",
				"KURTOSIS_INSPECT_ENVIRONMENT":      "/env/env.json",
			},
			expected: struct {
				fixTraefik      bool
				conductorConfig string
				environment     string
			}{
				fixTraefik:      true,
				conductorConfig: "/env/conductor.toml",
				environment:     "/env/env.json",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			for key, value := range tt.envVars {
				os.Setenv(key, value)
				defer os.Unsetenv(key)
			}

			app := &cli.App{
				Name:  "inspect",
				Flags: Flags,
				Action: func(ctx *cli.Context) error {
					assert.Equal(t, tt.expected.fixTraefik, ctx.Bool("fix-traefik"))
					assert.Equal(t, tt.expected.conductorConfig, ctx.String("conductor-config-path"))
					assert.Equal(t, tt.expected.environment, ctx.String("environment-path"))
					return nil
				},
			}

			err := app.Run(tt.args)
			require.NoError(t, err)
		})
	}
}

func TestFlagDefinitions(t *testing.T) {
	flagNames := make(map[string]bool)
	for _, flag := range Flags {
		for _, name := range flag.Names() {
			flagNames[name] = true
		}
	}

	assert.True(t, flagNames["fix-traefik"])
	assert.True(t, flagNames["conductor-config-path"])
	assert.True(t, flagNames["environment-path"])
	assert.True(t, flagNames["log.level"])
}

func TestEnvVarPrefix(t *testing.T) {
	assert.Equal(t, "KURTOSIS_INSPECT", EnvVarPrefix)
}

func TestFlagStructure(t *testing.T) {
	assert.NotEmpty(t, Flags)
	assert.Contains(t, optionalFlags, FixTraefik)
	assert.Contains(t, optionalFlags, ConductorConfig)
	assert.Contains(t, optionalFlags, Environment)
	assert.Empty(t, requiredFlags)
}
