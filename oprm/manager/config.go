package manager

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const DefaultConfigPath = ".oprm/config.yaml"

type GitHubRepoConfig struct {
	Owner        string `yaml:"owner,omitempty"`
	Repo         string `yaml:"repo,omitempty"`
	CheckoutPath string `yaml:"checkout_path,omitempty"`
}

type Config struct {
	ConfigPath   string           `yaml:"-"`
	MonorepoPath string           `yaml:"-"`
	RunsDir      string           `yaml:"runs_dir,omitempty"`
	BaseBranch   string           `yaml:"base_branch,omitempty"`
	GitHub       GitHubRepoConfig `yaml:"github,omitempty"`
	OpGeth       GitHubRepoConfig `yaml:"op_geth,omitempty"`
}

type ConfigOverrides struct {
	RunsDir            string
	BaseBranch         string
	GitHubOwner        string
	GitHubRepo         string
	OpGethOwner        string
	OpGethRepo         string
	OpGethCheckoutPath string
}

func DefaultConfig() *Config {
	return &Config{
		ConfigPath:   DefaultConfigPath,
		MonorepoPath: defaultMonorepoPath(),
		RunsDir:      ".oprm/releases",
		BaseBranch:   "develop",
		GitHub: GitHubRepoConfig{
			Owner: "ethereum-optimism",
			Repo:  "optimism",
		},
		OpGeth: GitHubRepoConfig{
			Owner:        "ethereum-optimism",
			Repo:         "op-geth",
			CheckoutPath: "../op-geth",
		},
	}
}

func LoadConfig(configPath string, overrides ConfigOverrides) (*Config, error) {
	cfg := DefaultConfig()
	explicitPath := configPath != ""
	if explicitPath {
		cfg.ConfigPath = configPath
	}

	data, err := os.ReadFile(cfg.ConfigPath)
	switch {
	case err == nil:
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("decode config file %q: %w", cfg.ConfigPath, err)
		}
	case os.IsNotExist(err) && !explicitPath:
		// Optional default config file.
	case err != nil:
		return nil, fmt.Errorf("read config file %q: %w", cfg.ConfigPath, err)
	}

	if overrides.RunsDir != "" {
		cfg.RunsDir = overrides.RunsDir
	}
	if overrides.BaseBranch != "" {
		cfg.BaseBranch = overrides.BaseBranch
	}
	if overrides.GitHubOwner != "" {
		cfg.GitHub.Owner = overrides.GitHubOwner
	}
	if overrides.GitHubRepo != "" {
		cfg.GitHub.Repo = overrides.GitHubRepo
	}
	if overrides.OpGethOwner != "" {
		cfg.OpGeth.Owner = overrides.OpGethOwner
	}
	if overrides.OpGethRepo != "" {
		cfg.OpGeth.Repo = overrides.OpGethRepo
	}
	if overrides.OpGethCheckoutPath != "" {
		cfg.OpGeth.CheckoutPath = overrides.OpGethCheckoutPath
	}
	cfg.RunsDir = filepath.Clean(cfg.RunsDir)
	cfg.MonorepoPath = defaultMonorepoPath()
	return cfg, nil
}
