package main

import (
	"os"

	"github.com/ethereum-optimism/optimism/ops-devnet/allocs"
	"github.com/ethereum-optimism/optimism/ops-devnet/utils"
	"github.com/ethereum-optimism/optimism/ops-devnet/flags"

	"github.com/ethereum/go-ethereum/log"
	"github.com/mattn/go-isatty"
	"github.com/urfave/cli/v2"
)

func main() {
	log.Root().SetHandler(log.StreamHandler(os.Stderr, log.TerminalFormat(isatty.IsTerminal(os.Stderr.Fd()))))

	app := &cli.App{
		Name:  "allocs",
		Usage: "Generates L1 allocs with genesis state.",
		Flags: flags.CommonFlags,
		Action: entrypoint,
	}

	if err := app.Run(os.Args); err != nil {
		log.Crit("allocs failed", "err", err)
	}
}

// entrypoint is the script entrypoint
func entrypoint(ctx *cli.Context) error {
	endpoint := ctx.String("l1-rpc-url")
	monorepo := ctx.String("monorepo-dir")
	if err := utils.MakeDirAll(utils.DevnetDirectory(monorepo)); err != nil {
		return err
	}

	stateDump, err := allocs.GenerateAllocs(monorepo, endpoint)
	if err != nil {
		return err
	}

	allocsPath := utils.AllocsJsonPath(monorepo)
	return utils.WriteJson(allocsPath, stateDump)
}
