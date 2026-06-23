package deployer

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-service/ctxinterrupt"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/urfave/cli/v2"
)

type PrepareConfig struct {
	Workdir string
	Logger  log.Logger
	// DeployerAddress is the account used to predict deployment addresses. It
	// does not need to be funded or signed for, since prepare does not broadcast.
	DeployerAddress common.Address
}

func (c *PrepareConfig) Check() error {
	if c.Workdir == "" {
		return fmt.Errorf("workdir must be specified")
	}

	if c.Logger == nil {
		return fmt.Errorf("logger must be specified")
	}

	if c.DeployerAddress == (common.Address{}) {
		return fmt.Errorf("deployer address must be specified")
	}

	return nil
}

func PrepareCLI() func(cliCtx *cli.Context) error {
	return func(cliCtx *cli.Context) error {
		logCfg := oplog.ReadCLIConfig(cliCtx)
		l := oplog.NewLogger(oplog.AppOut(cliCtx), logCfg)
		oplog.SetGlobalLogHandler(l.Handler())

		cfg, err := newPrepareConfig(cliCtx, l)
		if err != nil {
			return err
		}

		ctx := ctxinterrupt.WithCancelOnInterrupt(cliCtx.Context)

		return Prepare(ctx, cfg)
	}
}

// newPrepareConfig builds a PrepareConfig from the CLI flags. It parses and
// validates the deployer address here so a malformed value is rejected before
// it silently decodes to the zero address.
func newPrepareConfig(cliCtx *cli.Context, l log.Logger) (PrepareConfig, error) {
	deployerAddressRaw := cliCtx.String(DeployerAddressFlagName)
	if !common.IsHexAddress(deployerAddressRaw) {
		return PrepareConfig{}, fmt.Errorf("invalid deployer address: %q", deployerAddressRaw)
	}

	return PrepareConfig{
		Workdir:         cliCtx.String(WorkdirFlagName),
		Logger:          l,
		DeployerAddress: common.HexToAddress(deployerAddressRaw),
	}, nil
}

func Prepare(ctx context.Context, cfg PrepareConfig) error {
	if err := cfg.Check(); err != nil {
		return fmt.Errorf("invalid config for prepare: %w", err)
	}
	return nil
}
