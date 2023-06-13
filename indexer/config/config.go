package config

import (
	"os"

	"github.com/BurntSushi/toml"
)

// Config represents the `indexer.toml` file used to configure the indexer
type Config struct {
	Chain   ChainConfig
	RPCs    RPCsConfig `toml:"rpcs"`
	DB      DBConfig
	API     APIConfig
	Metrics MetricsConfig
}

// ChainConfig configures of the chain being indexed
type ChainConfig struct {
	// Configure known chains with the l2 chain id
	Preset int
}

// RPCsConfig configures the RPC urls
type RPCsConfig struct {
	L1RPC string `toml:"l1-rpc"`
	L2RPC string `toml:"l2-rpc"`
}

// DBConfig configures the postgres database
type DBConfig struct {
	Host     string
	Port     int
	User     string
	Password string
}

// APIConfig configures the API server
type APIConfig struct {
	Host string
	Port int
}

// MetricsConfig configures the metrics server
type MetricsConfig struct {
	Host string
	Port int
}

// LoadConfig loads the `indexer.toml` config file from a given path
func LoadConfig(path string) (Config, error) {
	var conf Config

	// Read the config file.
	data, err := os.ReadFile(path)
	if err != nil {
		return conf, err
	}

	// Replace environment variables.
	data = []byte(os.ExpandEnv(string(data)))

	// Decode the TOML data.
	if _, err := toml.Decode(string(data), &conf); err != nil {
		return conf, err
	}

	return conf, nil
}
