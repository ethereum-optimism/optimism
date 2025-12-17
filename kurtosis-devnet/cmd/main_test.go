package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ethereum-optimism/optimism/kurtosis-devnet/pkg/kurtosis"
	autofixTypes "github.com/ethereum-optimism/optimism/kurtosis-devnet/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

func TestParseFlags(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		wantCfg   *config
		wantError bool
	}{
		{
			name: "valid configuration",
			args: []string{
				"--template", "path/to/template.yaml",
				"--enclave", "test-enclave",
			},
			wantCfg: &config{
				templateFile:    "path/to/template.yaml",
				enclave:         "test-enclave",
				kurtosisPackage: kurtosis.DefaultPackageName,
			},
			wantError: false,
		},
		{
			name:      "missing required template",
			args:      []string{"--enclave", "test-enclave"},
			wantCfg:   nil,
			wantError: true,
		},
		{
			name: "with data file",
			args: []string{
				"--template", "path/to/template.yaml",
				"--data", "path/to/data.json",
			},
			wantCfg: &config{
				templateFile:    "path/to/template.yaml",
				dataFile:        "path/to/data.json",
				enclave:         kurtosis.DefaultEnclave,
				kurtosisPackage: kurtosis.DefaultPackageName,
			},
			wantError: false,
		},
		{
			name: "with autofix true",
			args: []string{
				"--template", "path/to/template.yaml",
				"--autofix", "true",
			},
			wantCfg: &config{
				templateFile:    "path/to/template.yaml",
				enclave:         kurtosis.DefaultEnclave,
				kurtosisPackage: kurtosis.DefaultPackageName,
				autofix:         "true",
			},
			wantError: false,
		},
		{
			name: "with autofix nuke",
			args: []string{
				"--template", "path/to/template.yaml",
				"--autofix", "nuke",
			},
			wantCfg: &config{
				templateFile:    "path/to/template.yaml",
				enclave:         kurtosis.DefaultEnclave,
				kurtosisPackage: kurtosis.DefaultPackageName,
				autofix:         "nuke",
			},
			wantError: false,
		},
		{
			name: "with invalid autofix value",
			args: []string{
				"--template", "path/to/template.yaml",
				"--autofix", "invalid",
			},
			wantCfg: &config{
				templateFile:    "path/to/template.yaml",
				enclave:         kurtosis.DefaultEnclave,
				kurtosisPackage: kurtosis.DefaultPackageName,
				autofix:         "invalid",
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cfg *config
			app := &cli.App{
				Flags: getFlags(),
				Action: func(c *cli.Context) (err error) {
					cfg, err = newConfig(c)
					return
				},
			}

			// Prepend program name to args as urfave/cli expects
			args := append([]string{"prog"}, tt.args...)

			err := app.Run(args)
			if tt.wantError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, cfg)
			assert.Equal(t, tt.wantCfg.templateFile, cfg.templateFile)
			assert.Equal(t, tt.wantCfg.enclave, cfg.enclave)
			assert.Equal(t, tt.wantCfg.kurtosisPackage, cfg.kurtosisPackage)
			if tt.wantCfg.dataFile != "" {
				assert.Equal(t, tt.wantCfg.dataFile, cfg.dataFile)
			}
			if tt.wantCfg.autofix != "" {
				assert.Equal(t, tt.wantCfg.autofix, cfg.autofix)
			}
		})
	}
}

func TestMainFuncValidatesConfig(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "main-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create test template
	templatePath := filepath.Join(tmpDir, "template.yaml")
	err = os.WriteFile(templatePath, []byte("name: test"), 0644)
	require.NoError(t, err)

	// Create environment output path
	envPath := filepath.Join(tmpDir, "env.json")

	app := &cli.App{
		Flags: getFlags(),
		Action: func(c *cli.Context) error {
			cfg, err := newConfig(c)
			if err != nil {
				return err
			}

			// Verify config values
			assert.Equal(t, templatePath, cfg.templateFile)
			assert.Equal(t, envPath, cfg.environment)
			assert.True(t, cfg.dryRun)

			// Create an empty environment file to simulate successful deployment
			return os.WriteFile(envPath, []byte("{}"), 0644)
		},
	}

	args := []string{
		"prog",
		"--template", templatePath,
		"--environment", envPath,
		"--dry-run",
	}

	err = app.Run(args)
	require.NoError(t, err)

	// Verify the environment file was created
	assert.FileExists(t, envPath)
}

func TestAutofixModes(t *testing.T) {
	tests := []struct {
		name         string
		autofixEnv   string
		autofixFlag  string
		expectedMode autofixTypes.AutofixMode
	}{
		{
			name:         "autofix disabled",
			autofixEnv:   "",
			autofixFlag:  "",
			expectedMode: autofixTypes.AutofixModeDisabled,
		},
		{
			name:         "autofix normal mode via env",
			autofixEnv:   "true",
			autofixFlag:  "",
			expectedMode: autofixTypes.AutofixModeNormal,
		},
		{
			name:         "autofix nuke mode via env",
			autofixEnv:   "nuke",
			autofixFlag:  "",
			expectedMode: autofixTypes.AutofixModeNuke,
		},
		{
			name:         "autofix normal mode via flag",
			autofixEnv:   "",
			autofixFlag:  "true",
			expectedMode: autofixTypes.AutofixModeNormal,
		},
		{
			name:         "autofix nuke mode via flag",
			autofixEnv:   "",
			autofixFlag:  "nuke",
			expectedMode: autofixTypes.AutofixModeNuke,
		},
		{
			name:         "flag takes precedence over env",
			autofixEnv:   "true",
			autofixFlag:  "nuke",
			expectedMode: autofixTypes.AutofixModeNuke,
		},
		{
			name:         "invalid autofix value",
			autofixEnv:   "invalid",
			autofixFlag:  "",
			expectedMode: autofixTypes.AutofixModeDisabled,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary directory for test files
			tmpDir, err := os.MkdirTemp("", "autofix-test")
			require.NoError(t, err)
			defer os.RemoveAll(tmpDir)

			// Create test template
			templatePath := filepath.Join(tmpDir, "template.yaml")
			err = os.WriteFile(templatePath, []byte("name: test"), 0644)
			require.NoError(t, err)

			// Create environment output path
			envPath := filepath.Join(tmpDir, "env.json")

			// Set up test environment
			if tt.autofixEnv != "" {
				t.Setenv("AUTOFIX", tt.autofixEnv)
			}

			app := &cli.App{
				Flags: getFlags(),
				Action: func(c *cli.Context) error {
					cfg, err := newConfig(c)
					if err != nil {
						return err
					}

					// Verify autofix mode
					autofixMode := autofixTypes.AutofixModeDisabled
					if cfg.autofix == "true" {
						autofixMode = autofixTypes.AutofixModeNormal
					} else if cfg.autofix == "nuke" {
						autofixMode = autofixTypes.AutofixModeNuke
					} else if os.Getenv("AUTOFIX") == "true" {
						autofixMode = autofixTypes.AutofixModeNormal
					} else if os.Getenv("AUTOFIX") == "nuke" {
						autofixMode = autofixTypes.AutofixModeNuke
					}
					assert.Equal(t, tt.expectedMode, autofixMode)

					// Create an empty environment file to simulate successful deployment
					return os.WriteFile(envPath, []byte("{}"), 0644)
				},
			}

			args := []string{
				"prog",
				"--template", templatePath,
				"--environment", envPath,
			}
			if tt.autofixFlag != "" {
				args = append(args, "--autofix", tt.autofixFlag)
			}

			err = app.Run(args)
			require.NoError(t, err)

			// Verify the environment file was created
			assert.FileExists(t, envPath)
		})
	}
}
