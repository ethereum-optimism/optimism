package config

import (
	"fmt"
	"os"
	"time"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"

	_ "gopkg.in/yaml.v3"
)

type Config struct {
	LogLevel string `yaml:"log_level"`
	DryRun   bool   `yaml:"dry_run"`

	Metrics MetricsConfig `yaml:"metrics"`
	Healthz HealthzConfig `yaml:"healthz"`

	PollInterval        time.Duration `yaml:"poll_interval"`
	NodeStateExpiration time.Duration `yaml:"node_state_expiration"`
	RPCTimeout          time.Duration `yaml:"rpc_timeout"`

	Nodes map[string]*NodeConfig `yaml:"nodes"`
}

type MetricsConfig struct {
	Enabled bool   `yaml:"enabled"`
	Debug   bool   `yaml:"debug"`
	Host    string `yaml:"host"`
	Port    string `yaml:"port"`
}

type HealthzConfig struct {
	Enabled bool   `yaml:"enabled"`
	Host    string `yaml:"host"`
	Port    string `yaml:"port"`
}

type NodeConfig struct {
	RPCAddress string `yaml:"rpc_address"`
}

func New(file string) (*Config, error) {
	cfg := &Config{}
	contents, err := os.ReadFile(file)
	if err != nil {
		fmt.Printf("error reading config file: %v\n", err)
	}
	if err := yaml.Unmarshal(contents, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) Validate() error {
	if c.Metrics.Enabled {
		if c.Metrics.Host == "" || c.Metrics.Port == "" {
			return errors.New("metrics is enabled but host or port are missing")
		}
	}
	if c.Healthz.Enabled {
		if c.Healthz.Host == "" || c.Healthz.Port == "" {
			return errors.New("healthz is enabled but host or port are missing")
		}
	}

	if len(c.Nodes) == 0 {
		return errors.New("no nodes configured")
	}

	for name, nodes := range c.Nodes {
		if nodes.RPCAddress == "" {
			return errors.Errorf("node [%s] rpc address is missing", name)
		}
	}

	if c.LogLevel == "" {
		c.LogLevel = "debug"
	}

	return nil
}
