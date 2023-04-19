package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	opnode "github.com/ethereum-optimism/optimism/op-node"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/sources"
	"github.com/ethereum-optimism/optimism/op-program/host/flags"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/params"
	"github.com/urfave/cli"
)

var (
	ErrMissingRollupConfig = errors.New("missing rollup config")
	ErrMissingL2Genesis    = errors.New("missing l2 genesis")
	ErrInvalidL1Head       = errors.New("invalid l1 head")
	ErrInvalidL2Head       = errors.New("invalid l2 head")
	ErrL1AndL2Inconsistent = errors.New("l1 and l2 options must be specified together or both omitted")
	ErrInvalidL2Claim      = errors.New("invalid l2 claim")
	ErrInvalidL2ClaimBlock = errors.New("invalid l2 claim block number")
	ErrDataDirRequired     = errors.New("datadir must be specified when in non-fetching mode")
)

type Config struct {
	Rollup *rollup.Config
	// DataDir is the directory to read/write pre-image data from/to.
	//If not set, an in-memory key-value store is used and fetching data must be enabled
	DataDir string

	// L1Head is the block has of the L1 chain head block
	L1Head     common.Hash
	L1URL      string
	L1TrustRPC bool
	L1RPCKind  sources.RPCProviderKind

	// L2Head is the agreed L2 block to start derivation from
	L2Head common.Hash
	L2URL  string
	// L2Claim is the claimed L2 output root to verify
	L2Claim common.Hash
	// L2ClaimBlockNumber is the block number the claimed L2 output root is from
	// Must be above 0 and to be a valid claim needs to be above the L2Head block.
	L2ClaimBlockNumber uint64
	// L2ChainConfig is the op-geth chain config for the L2 execution engine
	L2ChainConfig *params.ChainConfig
}

func (c *Config) Check() error {
	if c.Rollup == nil {
		return ErrMissingRollupConfig
	}
	if err := c.Rollup.Check(); err != nil {
		return err
	}
	if c.L1Head == (common.Hash{}) {
		return ErrInvalidL1Head
	}
	if c.L2Head == (common.Hash{}) {
		return ErrInvalidL2Head
	}
	if c.L2Claim == (common.Hash{}) {
		return ErrInvalidL2Claim
	}
	if c.L2ClaimBlockNumber == 0 {
		return ErrInvalidL2ClaimBlock
	}
	if c.L2ChainConfig == nil {
		return ErrMissingL2Genesis
	}
	if (c.L1URL != "") != (c.L2URL != "") {
		return ErrL1AndL2Inconsistent
	}
	if !c.FetchingEnabled() && c.DataDir == "" {
		return ErrDataDirRequired
	}
	return nil
}

func (c *Config) FetchingEnabled() bool {
	return c.L1URL != "" && c.L2URL != ""
}

// NewConfig creates a Config with all optional values set to the CLI default value
func NewConfig(rollupCfg *rollup.Config, l2Genesis *params.ChainConfig, l1Head common.Hash, l2Head common.Hash, l2Claim common.Hash, l2ClaimBlockNum uint64) *Config {
	return &Config{
		Rollup:             rollupCfg,
		L2ChainConfig:      l2Genesis,
		L1Head:             l1Head,
		L2Head:             l2Head,
		L2Claim:            l2Claim,
		L2ClaimBlockNumber: l2ClaimBlockNum,
		L1RPCKind:          sources.RPCKindBasic,
	}
}

func NewConfigFromCLI(ctx *cli.Context) (*Config, error) {
	if err := flags.CheckRequired(ctx); err != nil {
		return nil, err
	}
	rollupCfg, err := opnode.NewRollupConfig(ctx)
	if err != nil {
		return nil, err
	}
	l2Head := common.HexToHash(ctx.GlobalString(flags.L2Head.Name))
	if l2Head == (common.Hash{}) {
		return nil, ErrInvalidL2Head
	}
	l2Claim := common.HexToHash(ctx.GlobalString(flags.L2Claim.Name))
	if l2Claim == (common.Hash{}) {
		return nil, ErrInvalidL2Claim
	}
	l2ClaimBlockNum := ctx.GlobalUint64(flags.L2BlockNumber.Name)
	l1Head := common.HexToHash(ctx.GlobalString(flags.L1Head.Name))
	if l1Head == (common.Hash{}) {
		return nil, ErrInvalidL1Head
	}
	l2GenesisPath := ctx.GlobalString(flags.L2GenesisPath.Name)
	l2ChainConfig, err := loadChainConfigFromGenesis(l2GenesisPath)
	if err != nil {
		return nil, fmt.Errorf("invalid genesis: %w", err)
	}
	return &Config{
		Rollup:             rollupCfg,
		DataDir:            ctx.GlobalString(flags.DataDir.Name),
		L2URL:              ctx.GlobalString(flags.L2NodeAddr.Name),
		L2ChainConfig:      l2ChainConfig,
		L2Head:             l2Head,
		L2Claim:            l2Claim,
		L2ClaimBlockNumber: l2ClaimBlockNum,
		L1Head:             l1Head,
		L1URL:              ctx.GlobalString(flags.L1NodeAddr.Name),
		L1TrustRPC:         ctx.GlobalBool(flags.L1TrustRPC.Name),
		L1RPCKind:          sources.RPCProviderKind(ctx.GlobalString(flags.L1RPCProviderKind.Name)),
	}, nil
}

func loadChainConfigFromGenesis(path string) (*params.ChainConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read l2 genesis file: %w", err)
	}
	var genesis core.Genesis
	err = json.Unmarshal(data, &genesis)
	if err != nil {
		return nil, fmt.Errorf("parse l2 genesis file: %w", err)
	}
	return genesis.Config, nil
}
