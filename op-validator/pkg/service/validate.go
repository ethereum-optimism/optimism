package service

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/standard"

	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-validator/pkg/validations"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/urfave/cli/v2"
)

func ValidateCmd(cliCtx *cli.Context, release string) error {
	logCfg := oplog.ReadCLIConfig(cliCtx)
	lgr := oplog.NewLogger(oplog.AppOut(cliCtx), logCfg)
	cfg, err := NewConfig(cliCtx)
	if err != nil {
		return err
	}

	errors, err := Validate(cliCtx.Context, lgr, release, cfg)
	if err != nil {
		return fmt.Errorf("failed to validate: %w", err)
	}

	out := validations.Output{
		Errors: errors,
	}

	fmt.Println(out.AsMarkdown())

	if cliCtx.Bool(FailOnErrorFlag.Name) && len(errors) > 0 {
		return cli.Exit("Validation errors found", 1)
	}

	return nil
}

func Validate(ctx context.Context, lgr log.Logger, release string, cfg *Config) ([]string, error) {
	l1Client, err := rpc.Dial(cfg.L1RPCURL)
	if err != nil {
		return nil, fmt.Errorf("failed to dial L1 RPC: %w", err)
	}

	var validator validations.Validator
	switch release {
	case standard.ContractsV180Tag:
		validator = validations.NewV180Validator(l1Client)
	case standard.ContractsV200Tag:
		validator = validations.NewV200Validator(l1Client)
	case standard.ContractsV300Tag:
		validator = validations.NewV300Validator(l1Client)
	case standard.ContractsV400Tag:
		validator = validations.NewV400Validator(l1Client)
	default:
		return nil, fmt.Errorf("invalid release: %s", release)
	}

	return validator.Validate(ctx, validations.BaseValidatorInput{
		ProxyAdminAddress:   cfg.ProxyAdmin,
		SystemConfigAddress: cfg.SystemConfig,
		AbsolutePrestate:    cfg.AbsolutePrestate,
		L2ChainID:           cfg.L2ChainID,
	})
}
