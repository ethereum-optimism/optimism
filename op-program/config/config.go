package config

import (
	"errors"

	opnode "github.com/ethereum-optimism/optimism/op-node"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-program/flags"
	"github.com/ethereum/go-ethereum/common"
	"github.com/urfave/cli"
)

var (
	ErrMissingRollupConfig = errors.New("missing rollup config")
	ErrMissingL2Genesis    = errors.New("missing l2 genesis")
	ErrInvalidL2Head       = errors.New("invalid l2 head")
)

type Config struct {
	Rollup        *rollup.Config
	L2URL         string
	L2GenesisPath string
	L2Head        common.Hash
}

func (c *Config) Check() error {
	if c.Rollup == nil {
		return ErrMissingRollupConfig
	}
	if err := c.Rollup.Check(); err != nil {
		return err
	}
	if c.L2GenesisPath == "" {
		return ErrMissingL2Genesis
	}
	if c.L2Head == (common.Hash{}) {
		return ErrInvalidL2Head
	}
	return nil
}

func (c *Config) FetchingEnabled() bool {
	return c.L2URL != ""
}

// NewConfig creates a Config with all optional values set to the CLI default value
func NewConfig(rollupCfg *rollup.Config, l2GenesisPath string, l2Head common.Hash) *Config {
	return &Config{
		Rollup:        rollupCfg,
		L2GenesisPath: l2GenesisPath,
		L2Head:        l2Head,
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
	return &Config{
		Rollup:        rollupCfg,
		L2URL:         ctx.GlobalString(flags.L2NodeAddr.Name),
		L2GenesisPath: ctx.GlobalString(flags.L2GenesisPath.Name),
		L2Head:        l2Head,
	}, nil
}
