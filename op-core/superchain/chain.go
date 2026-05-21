package superchain

import (
	"archive/zip"
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path"
	"sort"
	"sync"

	"github.com/BurntSushi/toml"
	"github.com/klauspost/compress/zstd"
)

//go:embed superchain-configs.zip
var builtInConfigData []byte

var BuiltInConfigs *ChainConfigLoader

var Chains map[uint64]*Chain

var ErrUnknownChain = errors.New("unknown chain")

type ChainConfigLoader struct {
	configDataReader     fs.FS
	Chains               map[uint64]*Chain
	idsByName            map[string]uint64
	superchainsByNetwork map[string]Superchain
	mtx                  sync.Mutex
}

func NewChainConfigLoader(configData []byte) (*ChainConfigLoader, error) {
	configDataReader, err := zip.NewReader(bytes.NewReader(configData), int64(len(configData)))
	if err != nil {
		return nil, fmt.Errorf("opening zip reader: %w", err)
	}
	dictR, err := configDataReader.Open("dictionary")
	if err != nil {
		return nil, fmt.Errorf("error opening dictionary: %w", err)
	}
	defer dictR.Close()
	genesisZstdDict, err := io.ReadAll(dictR)
	if err != nil {
		return nil, fmt.Errorf("error reading dictionary: %w", err)
	}
	chainFile, err := configDataReader.Open("chains.json")
	if err != nil {
		return nil, fmt.Errorf("error opening chains file: %w", err)
	}
	defer chainFile.Close()
	chains := make(map[uint64]*Chain)
	if err := json.NewDecoder(chainFile).Decode(&chains); err != nil {
		return nil, fmt.Errorf("error decoding chains file: %w", err)
	}
	for _, chain := range chains {
		chain.configDataReader = configDataReader
		chain.genesisZstdDict = genesisZstdDict
	}

	idsByName := make(map[string]uint64)
	for chainID, chain := range chains {
		idsByName[chain.Name+"-"+chain.Network] = chainID
	}
	return &ChainConfigLoader{
		superchainsByNetwork: make(map[string]Superchain),
		configDataReader:     configDataReader,
		Chains:               chains,
		idsByName:            idsByName,
	}, nil
}

func ChainIDByName(name string) (uint64, error) {
	return BuiltInConfigs.ChainIDByName(name)
}

func (c *ChainConfigLoader) ChainIDByName(name string) (uint64, error) {
	id, ok := c.idsByName[name]
	if !ok {
		return 0, fmt.Errorf("%w %q", ErrUnknownChain, name)
	}
	return id, nil
}

func ChainNames() []string {
	return BuiltInConfigs.ChainNames()
}

func (c *ChainConfigLoader) ChainNames() []string {
	var out []string
	for _, ch := range c.Chains {
		out = append(out, ch.Name+"-"+ch.Network)
	}
	sort.Strings(out)
	return out
}

func GetChain(chainID uint64) (*Chain, error) {
	return BuiltInConfigs.GetChain(chainID)
}

func GetDepset(chainID uint64) (map[string]Dependency, error) {
	chain, err := BuiltInConfigs.GetChain(chainID)
	if err != nil {
		return nil, err
	}
	cfg, err := chain.Config()
	if err != nil {
		return nil, err
	}

	// depset of 1 (self) is the default when no dependencies are specified but interop_time is set
	if cfg.Interop == nil {
		cfg.Interop = &Interop{
			Dependencies: make(map[string]Dependency),
		}
		cfg.Interop.Dependencies[fmt.Sprintf("%d", cfg.ChainID)] = Dependency{}
	}

	return cfg.Interop.Dependencies, nil
}

func (c *ChainConfigLoader) GetChain(chainID uint64) (*Chain, error) {
	chain, ok := c.Chains[chainID]
	if !ok {
		return nil, fmt.Errorf("%w ID: %d", ErrUnknownChain, chainID)
	}
	return chain, nil
}

type Chain struct {
	Name    string `json:"name"`
	Network string `json:"network"`

	configDataReader fs.FS
	genesisZstdDict  []byte

	config  *ChainConfig
	genesis []byte

	// The config and genesis initialization is separated
	// to allow for lazy loading. Reading genesis files is
	// very expensive in Cannon so we only want to do it
	// when necessary.
	configOnce  sync.Once
	genesisOnce sync.Once
	err         error
}

func (c *Chain) Config() (*ChainConfig, error) {
	c.configOnce.Do(c.populateConfig)
	return c.config, c.err
}

func (c *Chain) GenesisData() ([]byte, error) {
	c.genesisOnce.Do(c.populateGenesis)
	return c.genesis, c.err
}

func (c *Chain) populateConfig() {
	configFile, err := c.configDataReader.Open(path.Join("configs", c.Network, c.Name+".toml"))
	if err != nil {
		c.err = fmt.Errorf("error opening chain config file %s/%s: %w", c.Network, c.Name, err)
		return
	}
	defer configFile.Close()

	var cfg ChainConfig
	if _, err := toml.NewDecoder(configFile).Decode(&cfg); err != nil {
		c.err = fmt.Errorf("error decoding chain config file %s/%s: %w", c.Network, c.Name, err)
		return
	}
	c.config = &cfg
}

func (c *Chain) populateGenesis() {
	genesisFile, err := c.configDataReader.Open(path.Join("genesis", c.Network, c.Name+".json.zst"))
	if err != nil {
		c.err = fmt.Errorf("error opening compressed genesis file %s/%s: %w", c.Network, c.Name, err)
		return
	}
	defer genesisFile.Close()
	zstdR, err := zstd.NewReader(genesisFile, zstd.WithDecoderDicts(c.genesisZstdDict))
	if err != nil {
		c.err = fmt.Errorf("error creating zstd reader for %s/%s: %w", c.Network, c.Name, err)
		return
	}
	defer zstdR.Close()

	out, err := io.ReadAll(zstdR)
	if err != nil {
		c.err = fmt.Errorf("error reading genesis file for %s/%s: %w", c.Network, c.Name, err)
		return
	}
	c.genesis = out
}

func init() {
	var err error
	BuiltInConfigs, err = NewChainConfigLoader(builtInConfigData)
	if err != nil {
		panic(err)
	}
	Chains = BuiltInConfigs.Chains
}
