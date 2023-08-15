package config

import (
	"os"

	"github.com/BurntSushi/toml"

	"github.com/ethereum-optimism/optimism/indexer/processor"
	"github.com/ethereum/go-ethereum/log"
	"github.com/joho/godotenv"
)

// Config represents the `indexer.toml` file used to configure the indexer
type Config struct {
	Chain   ChainConfig
	RPCs    RPCsConfig `toml:"rpcs"`
	DB      DBConfig
	API     APIConfig
	Metrics MetricsConfig
	Logger  log.Logger `toml:"-"`
}

// ChainConfig configures of the chain being indexed
type ChainConfig struct {
	// Configure known chains with the l2 chain id
	// NOTE - This currently performs no lookups to extract known L1 contracts by l2 chain id
	Preset      int
	L1Contracts processor.L1Contracts `toml:"l1-contracts"`
	// L1StartingHeight is the block height to start indexing from
	// NOTE - This is currently unimplemented
	L1StartingHeight int
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
	Name     string
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
	if err := godotenv.Load(); err != nil {
		log.Warn("Unable to load .env file", err)
		log.Info("Continuing without .env file")
	} else {
		log.Info("Loaded .env file")
	}

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
		log.Info("Failed to decode config file", "message", err)
		return conf, err
	}

	log.Debug("Loaded config file", conf)

	return conf, nil
}
