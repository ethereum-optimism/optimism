package config

import (
	"fmt"
	"math/big"
	"os"
	"reflect"

	"github.com/BurntSushi/toml"
	"github.com/ethereum/go-ethereum/common"
	geth_log "github.com/ethereum/go-ethereum/log"
)

// in future presets can just be onchain config and fetched on initialization

// Config represents the `indexer.toml` file used to configure the indexer
type Config struct {
	Chain   ChainConfig   `toml:"chain"`
	RPCs    RPCsConfig    `toml:"rpcs"`
	DB      DBConfig      `toml:"db"`
	API     APIConfig     `toml:"api"`
	Metrics MetricsConfig `toml:"metrics"`
}

// fetch this via onchain config from RPCsConfig and remove from config in future
type L1Contracts struct {
	OptimismPortalProxy         common.Address `toml:"optimism-portal"`
	L2OutputOracleProxy         common.Address `toml:"l2-output-oracle"`
	L1CrossDomainMessengerProxy common.Address `toml:"l1-cross-domain-messenger"`
	L1StandardBridgeProxy       common.Address `toml:"l1-standard-bridge"`

	// Some more contracts -- L1ERC721Bridge, ProxyAdmin, SystemConfig, etc
	// Ignore the auxiliary contracts?

	// Legacy contracts? We'll add this in to index the legacy chain.
	// Remove afterwards?
}

// converts struct of to a slice of addresses for easy iteration
// also validates that all fields are addresses
func (c *L1Contracts) AsSlice() ([]common.Address, error) {
	clone := *c
	contractValue := reflect.ValueOf(clone)
	fields := reflect.VisibleFields(reflect.TypeOf(clone))
	l1Contracts := make([]common.Address, len(fields))
	for i, field := range fields {
		// ruleid: unsafe-reflect-by-name
		addr, ok := (contractValue.FieldByName(field.Name).Interface()).(common.Address)
		if !ok {
			return nil, fmt.Errorf("non-address found in L1Contracts: %s", field.Name)
		}

		l1Contracts[i] = addr
	}

	return l1Contracts, nil
}

// ChainConfig configures of the chain being indexed
type ChainConfig struct {
	// Configure known chains with the l2 chain id
	// NOTE - This currently performs no lookups to extract known L1 contracts by l2 chain id
	Preset      int
	L1Contracts L1Contracts `toml:"l1-contracts"`
	// L1StartingHeight is the block height to start indexing from
	L1StartingHeight uint `toml:"l1-starting-height"`
}

func (cc *ChainConfig) L1StartHeight() *big.Int {
	return big.NewInt(int64(cc.L1StartingHeight))
}

// RPCsConfig configures the RPC urls
type RPCsConfig struct {
	L1RPC string `toml:"l1-rpc"`
	L2RPC string `toml:"l2-rpc"`
}

// DBConfig configures the postgres database
type DBConfig struct {
	Host     string `toml:"host"`
	Port     int    `toml:"port"`
	Name     string `toml:"name"`
	User     string `toml:"user"`
	Password string `toml:"password"`
}

// APIConfig configures the API server
type APIConfig struct {
	Host string `toml:"host"`
	Port int    `toml:"port"`
}

// MetricsConfig configures the metrics server
type MetricsConfig struct {
	Host string `toml:"host"`
	Port int    `toml:"port"`
}

// LoadConfig loads the `indexer.toml` config file from a given path
func LoadConfig(logger geth_log.Logger, path string) (Config, error) {
	logger.Info("Loading config file", "path", path)
	var conf Config

	data, err := os.ReadFile(path)
	if err != nil {
		return conf, err
	}

	data = []byte(os.ExpandEnv(string(data)))

	logger.Debug("Decoding config file", "data", string(data))

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
