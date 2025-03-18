package compare

import (
	"context"
	"fmt"

	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum/go-ethereum/log"
	"github.com/urfave/cli/v2"
)

func CompareCLI() func(ctx *cli.Context) error {
	return func(cliCtx *cli.Context) error {
		logCfg := oplog.ReadCLIConfig(cliCtx)
		lgr := oplog.NewLogger(oplog.AppOut(cliCtx), logCfg)
		cfg, err := NewCompareConfig(cliCtx)
		if err != nil {
			return err
		}

		diffs, err := Compare(cliCtx.Context, lgr, cfg)
		if err != nil {
			return fmt.Errorf("failed to validate: %w", err)
		}

		fmt.Printf("diffs: %s\n", FormatDiffs(diffs))
		lgr.Info("completed comparing chain info")
		return nil
	}
}

func Compare(ctx context.Context, lgr log.Logger, cfg *CompareConfig) (map[uint64]ChainDiff, error) {
	diffs, err := CompareAddresses(lgr, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to compare addresses: %w", err)
	}

	return diffs, nil
}
