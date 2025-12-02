package inspect

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

func TestNewConfig(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected *Config
		wantErr  bool
	}{
		{
			name: "valid config",
			args: []string{"inspect", "test-enclave"},
			expected: &Config{
				EnclaveID:           "test-enclave",
				FixTraefik:          false,
				ConductorConfigPath: "",
				EnvironmentPath:     "",
			},
			wantErr: false,
		},
		{
			name: "config with flags",
			args: []string{
				"inspect",
				"--fix-traefik",
				"--conductor-config-path", "/tmp/conductor.toml",
				"--environment-path", "/tmp/env.json",
				"my-enclave",
			},
			expected: &Config{
				EnclaveID:           "my-enclave",
				FixTraefik:          true,
				ConductorConfigPath: "/tmp/conductor.toml",
				EnvironmentPath:     "/tmp/env.json",
			},
			wantErr: false,
		},
		{
			name:     "no arguments",
			args:     []string{"inspect"},
			expected: nil,
			wantErr:  true,
		},
		{
			name:     "too many arguments",
			args:     []string{"inspect", "enclave1", "enclave2"},
			expected: nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := &cli.App{
				Name: "inspect",
				Flags: []cli.Flag{
					&cli.BoolFlag{Name: "fix-traefik"},
					&cli.StringFlag{Name: "conductor-config-path"},
					&cli.StringFlag{Name: "environment-path"},
				},
				Action: func(ctx *cli.Context) error {
					cfg, err := NewConfig(ctx)

					if tt.wantErr {
						assert.Error(t, err)
						assert.Nil(t, cfg)
					} else {
						require.NoError(t, err)
						require.NotNil(t, cfg)
						assert.Equal(t, tt.expected.EnclaveID, cfg.EnclaveID)
						assert.Equal(t, tt.expected.FixTraefik, cfg.FixTraefik)
						assert.Equal(t, tt.expected.ConductorConfigPath, cfg.ConductorConfigPath)
						assert.Equal(t, tt.expected.EnvironmentPath, cfg.EnvironmentPath)
					}
					return nil
				},
			}

			err := app.Run(tt.args)
			require.NoError(t, err)
		})
	}
}
