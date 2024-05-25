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

// Configures the server
type ServerConfig struct {
	Host         string `toml:"host"`
	Port         int    `toml:"port"`
	WriteTimeout int    `toml:"timeout"`
}

// LoadConfig loads the `indexer.toml` config file from a given path
func LoadConfig(log log.Logger, path string) (Config, error) {
	log.Debug("loading config", "path", path)

	var cfg Config
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, err
	}

	data = []byte(os.ExpandEnv(string(data)))
	log.Debug("parsed config file", "data", string(data))

	md, err := toml.Decode(string(data), &cfg)
	if err != nil {
		log.Error("failed to decode config file", "err", err)
		return cfg, err
	}

	if len(md.Undecoded()) > 0 {
		log.Error("unknown fields in config file", "fields", md.Undecoded())
		err = fmt.Errorf("unknown fields in config file: %v", md.Undecoded())
		return cfg, err
	}

	if cfg.Chain.Preset == DevnetPresetId {
		preset, err := DevnetPreset()
		if err != nil {
			return cfg, err
		}

		log.Info("detected preset", "preset", DevnetPresetId, "name", preset.Name)
		cfg.Chain = preset.ChainConfig
	} else if cfg.Chain.Preset != 0 {
		preset, ok := Presets[cfg.Chain.Preset]
		if !ok {
			return cfg, fmt.Errorf("unknown preset: %d", cfg.Chain.Preset)
		}

		log.Info("detected preset", "preset", cfg.Chain.Preset, "name", preset.Name)
		cfg.Chain = preset.ChainConfig
	}

	// Setup L2Contracts from predeploys
	cfg.Chain.L2Contracts = L2ContractsFromPredeploys()

	// Deserialize the config file again when a preset is configured such that
	// precedence is given to the config file vs the preset
	if cfg.Chain.Preset > 0 {
		if _, err := toml.Decode(string(data), &cfg); err != nil {
			log.Error("failed to decode config file", "err", err)
			return cfg, err
		}
	}

	// Defaults for any unset options

	if cfg.Chain.L1PollingInterval == 0 {
		cfg.Chain.L1PollingInterval = defaultLoopInterval
	}

	if cfg.Chain.L2PollingInterval == 0 {
		cfg.Chain.L2PollingInterval = defaultLoopInterval
	}

	if cfg.Chain.L1HeaderBufferSize == 0 {
		cfg.Chain.L1HeaderBufferSize = defaultHeaderBufferSize
	}

	if cfg.Chain.L2HeaderBufferSize == 0 {
		cfg.Chain.L2HeaderBufferSize = defaultHeaderBufferSize
	}

	log.Info("loaded chain config", "config", cfg.Chain)
	return cfg, nil
}
