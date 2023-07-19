package config

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/urfave/cli/v2"

	"github.com/ethereum-optimism/optimism/op-challenger/flags"
	opservice "github.com/ethereum-optimism/optimism/op-service"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
)

var (
	ErrMissingL1EthRPC      = errors.New("missing l1 eth rpc url")
	ErrMissingGameAddress   = errors.New("missing game address")
	ErrMissingAlphabetTrace = errors.New("missing alphabet trace")
)

// Config is a well typed config that is parsed from the CLI params.
// This also contains config options for auxiliary services.
// It is used to initialize the challenger.
type Config struct {
	L1EthRpc                string         // L1 RPC Url
	GameAddress             common.Address // Address of the fault game
	AlphabetTrace           string         // String for the AlphabetTraceProvider
	AgreeWithProposedOutput bool           // Temporary config if we agree or disagree with the posted output
	GameDepth               int            // Depth of the game tree

	TxMgrConfig txmgr.CLIConfig
}

func NewConfig(
	l1EthRpc string,
	GameAddress common.Address,
	AlphabetTrace string,
	AgreeWithProposedOutput bool,
	GameDepth int,
) Config {
	return Config{
		L1EthRpc:                l1EthRpc,
		GameAddress:             GameAddress,
		AlphabetTrace:           AlphabetTrace,
		TxMgrConfig:             txmgr.NewCLIConfig(l1EthRpc),
		AgreeWithProposedOutput: AgreeWithProposedOutput,
		GameDepth:               GameDepth,
	}
}

func (c Config) Check() error {
	if c.L1EthRpc == "" {
		return ErrMissingL1EthRPC
	}
	if c.GameAddress == (common.Address{}) {
		return ErrMissingGameAddress
	}
	if c.AlphabetTrace == "" {
		return ErrMissingAlphabetTrace
	}
	if err := c.TxMgrConfig.Check(); err != nil {
		return err
	}
	return nil
}

// NewConfigFromCLI parses the Config from the provided flags or environment variables.
func NewConfigFromCLI(ctx *cli.Context) (*Config, error) {
	if err := flags.CheckRequired(ctx); err != nil {
		return nil, err
	}
	dgfAddress, err := opservice.ParseAddress(ctx.String(flags.DGFAddressFlag.Name))
	if err != nil {
		return nil, err
	}

	txMgrConfig := txmgr.ReadCLIConfig(ctx)

	return &Config{
		// Required Flags
		L1EthRpc:                ctx.String(flags.L1EthRpcFlag.Name),
		GameAddress:             dgfAddress,
		AlphabetTrace:           ctx.String(flags.AlphabetFlag.Name),
		AgreeWithProposedOutput: ctx.Bool(flags.AgreeWithProposedOutputFlag.Name),
		GameDepth:               ctx.Int(flags.GameDepthFlag.Name),
		TxMgrConfig:             txMgrConfig,
	}, nil
}
