package config

import (
	"fmt"
	"os"
	"reflect"

	"github.com/BurntSushi/toml"
	"github.com/ethereum-optimism/optimism/op-bindings/predeploys"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

const (
	// default to 5 seconds
	defaultLoopInterval     = 5000
	defaultHeaderBufferSize = 500
)

// In the future, presets can just be onchain config and fetched on initialization

// Config represents the `indexer.toml` file used to configure the indexer
type Config struct {
	Chain         ChainConfig  `toml:"chain"`
	RPCs          RPCsConfig   `toml:"rpcs"`
	DB            DBConfig     `toml:"db"`
	HTTPServer    ServerConfig `toml:"http"`
	MetricsServer ServerConfig `toml:"metrics"`
}

// L1Contracts configures deployed contracts
type L1Contracts struct {
	// administrative
	AddressManager    common.Address `toml:"address-manager"`
	SystemConfigProxy common.Address `toml:"system-config"`

	// rollup state
	OptimismPortalProxy common.Address `toml:"optimism-portal"`
	L2OutputOracleProxy common.Address `toml:"l2-output-oracle"`

	// bridging
	L1CrossDomainMessengerProxy common.Address `toml:"l1-cross-domain-messenger"`
	L1StandardBridgeProxy       common.Address `toml:"l1-standard-bridge"`
	L1ERC721BridgeProxy         common.Address `toml:"l1-erc721-bridge"`

	// IGNORE: legacy contracts (only settable via presets)
	LegacyCanonicalTransactionChain common.Address `toml:"-"`
	LegacyStateCommitmentChain      common.Address `toml:"-"`
}

func (c L1Contracts) ForEach(cb func(string, common.Address) error) error {
	contracts := reflect.ValueOf(c)
	fields := reflect.VisibleFields(reflect.TypeOf(c))
	for _, field := range fields {
		// ruleid: unsafe-reflect-by-name
		addr := (contracts.FieldByName(field.Name).Interface()).(common.Address)
		if err := cb(field.Name, addr); err != nil {
			return err
		}
	}

	return nil
}

// L2Contracts configures core predeploy contracts. We explicitly specify
// fields until we can detect and backfill new addresses
type L2Contracts struct {
	L2ToL1MessagePasser    common.Address
	L2CrossDomainMessenger common.Address
	L2StandardBridge       common.Address
	L2ERC721Bridge         common.Address
}

func L2ContractsFromPredeploys() L2Contracts {
	return L2Contracts{
		L2ToL1MessagePasser:    predeploys.L2ToL1MessagePasserAddr,
		L2CrossDomainMessenger: predeploys.L2CrossDomainMessengerAddr,
		L2StandardBridge:       predeploys.L2StandardBridgeAddr,
		L2ERC721Bridge:         predeploys.L2ERC721BridgeAddr,
	}
}

func (c L2Contracts) ForEach(cb func(string, common.Address) error) error {
	contracts := reflect.ValueOf(c)
	fields := reflect.VisibleFields(reflect.TypeOf(c))
	for _, field := range fields {
		// ruleid: unsafe-reflect-by-name
		addr := (contracts.FieldByName(field.Name).Interface()).(common.Address)
		if err := cb(field.Name, addr); err != nil {
			return err
		}
	}

	return nil
}

// ChainConfig configures of the chain being indexed
type ChainConfig struct {
	// Configure known chains with the l2 chain id
	Preset           int
	L1StartingHeight uint `toml:"l1-starting-height"`

	L1Contracts L1Contracts `toml:"l1-contracts"`
	L2Contracts L2Contracts `toml:"-"`

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
		preset, ok := Presets[conf.Chain.Preset]
		if !ok {
			return conf, fmt.Errorf("unknown preset: %d", conf.Chain.Preset)
		}

		log.Info("detected preset", "preset", conf.Chain.Preset, "name", preset.Name)
		log.Info("setting L1 information from preset")
		conf.Chain.L1Contracts = preset.ChainConfig.L1Contracts
		conf.Chain.L1StartingHeight = preset.ChainConfig.L1StartingHeight
		conf.Chain.L1BedrockStartingHeight = preset.ChainConfig.L1BedrockStartingHeight
		conf.Chain.L2BedrockStartingHeight = preset.ChainConfig.L1BedrockStartingHeight
	}

	// Setup L2Contracts from predeploys
	conf.Chain.L2Contracts = L2ContractsFromPredeploys()

	// Setup defaults for some unset options

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
