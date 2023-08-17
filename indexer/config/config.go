package config

import (
	"fmt"
	"os"
	"reflect"

	"github.com/BurntSushi/toml"
	"github.com/ethereum/go-ethereum/common"
	geth_log "github.com/ethereum/go-ethereum/log"
	"github.com/joho/godotenv"
)

// in future presets can just be onchain config and fetched on initialization

// Config represents the `indexer.toml` file used to configure the indexer
type Config struct {
	Chain   ChainConfig
	RPCs    RPCsConfig `toml:"rpcs"`
	DB      DBConfig
	API     APIConfig
	Metrics MetricsConfig
}

// fetch this via onchain config from RPCsConfig and remove from config in future
type L1Contracts struct {
	OptimismPortal         common.Address `toml:"optimism-portal"`
	L2OutputOracle         common.Address `toml:"l2-output-oracle"`
	L1CrossDomainMessenger common.Address `toml:"l1-cross-domain-messenger"`
	L1StandardBridge       common.Address `toml:"l1-standard-bridge"`
	L1ERC721Bridge         common.Address `toml:"l1-erc721-bridge"`

	// Some more contracts -- ProxyAdmin, SystemConfig, etcc
	// Ignore the auxiliary contracts?

	// Legacy contracts? We'll add this in to index the legacy chain.
	// Remove afterwards?
}

func (c L1Contracts) ToSlice() []common.Address {
	fields := reflect.VisibleFields(reflect.TypeOf(c))
	v := reflect.ValueOf(c)

	contracts := make([]common.Address, len(fields))
	for i, field := range fields {
		contracts[i] = (v.FieldByName(field.Name).Interface()).(common.Address)
	}

	return contracts
}

// ChainConfig configures of the chain being indexed
type ChainConfig struct {
	// Configure known chains with the l2 chain id
	// NOTE - This currently performs no lookups to extract known L1 contracts by l2 chain id
	Preset      int
	L1Contracts L1Contracts `toml:"l1-contracts"`
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
func LoadConfig(logger geth_log.Logger, path string) (Config, error) {
	if err := godotenv.Load(); err != nil {
		logger.Warn("Unable to load .env file", err)
		logger.Info("Continuing without .env file")
	} else {
		logger.Info("Loaded .env file")
	}

	var conf Config

	data, err := os.ReadFile(path)
	if err != nil {
		return conf, err
	}

	data = []byte(os.ExpandEnv(string(data)))

	if _, err := toml.Decode(string(data), &conf); err != nil {
		logger.Info("Failed to decode config file", "message", err)
		return conf, err
	}

	if conf.Chain.Preset != 0 {
		knownContracts, ok := presetL1Contracts[conf.Chain.Preset]
		if ok {
			conf.Chain.L1Contracts = knownContracts
		} else {
			return conf, fmt.Errorf("unknown preset: %d", conf.Chain.Preset)
		}
	}

	logger.Debug("Loaded config file", conf)

	return conf, nil
}
