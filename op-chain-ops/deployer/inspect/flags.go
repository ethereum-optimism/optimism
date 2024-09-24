package inspect

import (
	"fmt"

	op_service "github.com/ethereum-optimism/optimism/op-service"

	"github.com/ethereum-optimism/optimism/op-chain-ops/deployer"
	"github.com/ethereum/go-ethereum/common"
	"github.com/urfave/cli/v2"
)

const (
	OutfileFlagName = "outfile"
)

var (
	FlagOutfile = &cli.StringFlag{
		Name:  OutfileFlagName,
		Usage: "output file. set to - to use stdout",
		Value: "-",
	}
)

var Flags = []cli.Flag{
	deployer.WorkdirFlag,
	FlagOutfile,
}

var Commands = []*cli.Command{
	{
		Name:      "genesis",
		Usage:     "outputs the genesis for an L2 chain",
		Args:      true,
		ArgsUsage: "<chain-id>",
		Action:    GenesisCLI,
		Flags:     Flags,
	},
	{
		Name:      "rollup",
		Usage:     "outputs the rollup config for an L2 chain",
		Args:      true,
		ArgsUsage: "<chain-id>",
		Action:    RollupCLI,
		Flags:     Flags,
	},
}

type cliConfig struct {
	Workdir string
	Outfile string
	ChainID common.Hash
}

func readConfig(cliCtx *cli.Context) (cliConfig, error) {
	var cfg cliConfig

	outfile := cliCtx.String(OutfileFlagName)
	if outfile == "" {
		return cfg, fmt.Errorf("outfile flag is required")
	}

	workdir := cliCtx.String(deployer.WorkdirFlagName)
	if workdir == "" {
		return cfg, fmt.Errorf("workdir flag is required")
	}

	chainIDStr := cliCtx.Args().First()
	if chainIDStr == "" {
		return cfg, fmt.Errorf("chain-id argument is required")
	}

	chainID, err := op_service.Parse256BitChainID(chainIDStr)
	if err != nil {
		return cfg, fmt.Errorf("failed to parse chain ID: %w", err)
	}

	return cliConfig{
		Workdir: cliCtx.String(deployer.WorkdirFlagName),
		Outfile: cliCtx.String(OutfileFlagName),
		ChainID: chainID,
	}, nil
}
