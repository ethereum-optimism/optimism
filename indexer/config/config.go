package config

import (
	"fmt"
	"os"
	"reflect"

	"github.com/BurntSushi/toml"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

const (
	// default to 5 seconds
	defaultLoopInterval     = 5000
	defaultHeaderBufferSize = 500
)

// in future presets can just be onchain config and fetched on initialization

// Config represents the `indexer.toml` file used to configure the indexer
type Config struct {
	Chain         ChainConfig  `toml:"chain"`
	RPCs          RPCsConfig   `toml:"rpcs"`
	DB            DBConfig     `toml:"db"`
	HTTPServer    ServerConfig `toml:"http"`
	MetricsServer ServerConfig `toml:"metrics"`
}

// fetch this via onchain config from RPCsConfig and remove from config in future
type L1Contracts struct {
	OptimismPortalProxy         common.Address `toml:"optimism-portal"`
	L2OutputOracleProxy         common.Address `toml:"l2-output-oracle"`
	L1CrossDomainMessengerProxy common.Address `toml:"l1-cross-domain-messenger"`
	L1StandardBridgeProxy       common.Address `toml:"l1-standard-bridge"`

	// Pre-Bedrock Legacy Contracts
	LegacyCanonicalTransactionChain common.Address `toml:"l1-canonical-transaction-chain"`

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
	Preset int

	L1Contracts      L1Contracts `toml:"l1-contracts"`
	L1StartingHeight uint        `toml:"l1-starting-height"`

	// Bedrock starting heights only applicable for OP-Mainnet & OP-Goerli
	L1BedrockStartingHeight uint `toml:"-"`
	L2BedrockStartingHeight uint `toml:"-"`

	// These configuration options will be removed once
	// native reorg handling is implemented
	L1ConfirmationDepth uint `toml:"l1-confirmation-depth"`
	L2ConfirmationDepth uint `toml:"l2-confirmation-depth"`

	L1PollingInterval uint `toml:"l1-polling-interval"`
	L2PollingInterval uint `toml:"l2-polling-interval"`

	L1HeaderBufferSize uint `toml:"l1-header-buffer-size"`
	L2HeaderBufferSize uint `toml:"l2-header-buffer-size"`
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

// Configures the a server
type ServerConfig struct {
	Host string `toml:"host"`
	Port int    `toml:"port"`
}

// LoadConfig loads the `indexer.toml` config file from a given path
func LoadConfig(log log.Logger, path string) (Config, error) {
	log.Debug("loading config", "path", path)

	var conf Config
	data, err := os.ReadFile(path)
	if err != nil {
		return conf, err
	}

	data = []byte(os.ExpandEnv(string(data)))
	log.Debug("parsed config file", "data", string(data))
	if _, err := toml.Decode(string(data), &conf); err != nil {
		log.Info("failed to decode config file", "err", err)
		return conf, err
	}

	if conf.Chain.Preset != 0 {
		knownPreset, ok := presetConfigs[conf.Chain.Preset]
		if !ok {
			return conf, fmt.Errorf("unknown preset: %d", conf.Chain.Preset)
		}

		conf.Chain.L1Contracts = knownPreset.L1Contracts
		conf.Chain.L1StartingHeight = knownPreset.L1StartingHeight
		conf.Chain.L1BedrockStartingHeight = knownPreset.L1BedrockStartingHeight
		conf.Chain.L2BedrockStartingHeight = knownPreset.L1BedrockStartingHeight
	}

	// Set polling defaults if not set
	if conf.Chain.L1PollingInterval == 0 {
		log.Info("setting default L1 polling interval", "interval", defaultLoopInterval)
		conf.Chain.L1PollingInterval = defaultLoopInterval
	}

	if conf.Chain.L2PollingInterval == 0 {
		log.Info("setting default L2 polling interval", "interval", defaultLoopInterval)
		conf.Chain.L2PollingInterval = defaultLoopInterval
	}

	if conf.Chain.L1HeaderBufferSize == 0 {
		log.Info("setting default L1 header buffer", "size", defaultHeaderBufferSize)
		conf.Chain.L1HeaderBufferSize = defaultHeaderBufferSize
	}

	if conf.Chain.L2HeaderBufferSize == 0 {
		log.Info("setting default L2 header buffer", "size", defaultHeaderBufferSize)
		conf.Chain.L2HeaderBufferSize = defaultHeaderBufferSize
	}

	log.Info("loaded config")
	return conf, nil
}
