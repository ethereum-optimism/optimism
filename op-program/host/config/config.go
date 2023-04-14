package config

import (
	"errors"

	opnode "github.com/ethereum-optimism/optimism/op-node"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/sources"
	"github.com/ethereum-optimism/optimism/op-program/host/flags"
	"github.com/ethereum/go-ethereum/common"
	"github.com/urfave/cli"
)

var (
	ErrMissingRollupConfig = errors.New("missing rollup config")
	ErrMissingL2Genesis    = errors.New("missing l2 genesis")
	ErrInvalidL1Head       = errors.New("invalid l1 head")
	ErrInvalidL2Head       = errors.New("invalid l2 head")
	ErrL1AndL2Inconsistent = errors.New("l1 and l2 options must be specified together or both omitted")
	ErrInvalidL2Claim      = errors.New("invalid l2 claim")
)

type Config struct {
	Rollup        *rollup.Config
	L2URL         string
	L2GenesisPath string
	L1Head        common.Hash
	L2Head        common.Hash
	L2Claim       common.Hash
	L1URL         string
	L1TrustRPC    bool
	L1RPCKind     sources.RPCProviderKind
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
	if c.L2GenesisPath == "" {
		return ErrMissingL2Genesis
	}
	if (c.L1URL != "") != (c.L2URL != "") {
		return ErrL1AndL2Inconsistent
	}
	return nil
}

func (c *Config) FetchingEnabled() bool {
	return c.L1URL != "" && c.L2URL != ""
}

// NewConfig creates a Config with all optional values set to the CLI default value
func NewConfig(rollupCfg *rollup.Config, l2GenesisPath string, l1Head common.Hash, l2Head common.Hash, l2Claim common.Hash) *Config {
	return &Config{
		Rollup:        rollupCfg,
		L2GenesisPath: l2GenesisPath,
		L1Head:        l1Head,
		L2Head:        l2Head,
		L2Claim:       l2Claim,
		L1RPCKind:     sources.RPCKindBasic,
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
	l1Head := common.HexToHash(ctx.GlobalString(flags.L1Head.Name))
	if l1Head == (common.Hash{}) {
		return nil, ErrInvalidL1Head
	}
	return &Config{
		Rollup:        rollupCfg,
		L2URL:         ctx.GlobalString(flags.L2NodeAddr.Name),
		L2GenesisPath: ctx.GlobalString(flags.L2GenesisPath.Name),
		L2Head:        l2Head,
		L2Claim:       l2Claim,
		L1Head:        l1Head,
		L1URL:         ctx.GlobalString(flags.L1NodeAddr.Name),
		L1TrustRPC:    ctx.GlobalBool(flags.L1TrustRPC.Name),
		L1RPCKind:     sources.RPCProviderKind(ctx.GlobalString(flags.L1RPCProviderKind.Name)),
	}, nil
}
