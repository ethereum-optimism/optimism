package main

import (
	"os"
	"errors"

	"github.com/ethereum-optimism/optimism/ops-devnet/allocs"
	"github.com/ethereum-optimism/optimism/ops-devnet/genesis"
	"github.com/ethereum-optimism/optimism/ops-devnet/devnet"
	"github.com/ethereum-optimism/optimism/ops-devnet/utils"
	"github.com/ethereum-optimism/optimism/ops-devnet/flags"

	"github.com/ethereum/go-ethereum/log"
	"github.com/mattn/go-isatty"
	"github.com/urfave/cli/v2"
)

func main() {
	log.Root().SetHandler(log.StreamHandler(os.Stderr, log.TerminalFormat(isatty.IsTerminal(os.Stderr.Fd()))))

	app := &cli.App{
		Name:  "devnet",
		Usage: "Brings up a local devnet with deployed contracts.",
		Flags: flags.CommonFlags,
		Action: entrypoint,
	}

	if err := app.Run(os.Args); err != nil {
		log.Crit("devnet failed", "err", err)
	}
}

// entrypoint is the script entrypoint
func entrypoint(ctx *cli.Context) error {
	endpoint := ctx.String("l1-rpc-url")
	l2Endpoint := ctx.String("l2-rpc-url")
	monorepo := ctx.String("monorepo-dir")
	if err := utils.MakeDirAll(utils.DevnetDirectory(monorepo)); err != nil {
		return err
	}

	allocsPath := utils.AllocsJsonPath(monorepo)
	genesisPath := utils.GenesisJsonPath(monorepo)
	genesisL2Path := utils.GenesisL2JsonPath(monorepo)

	// Generate Genesis if it doesn't exist
	if _, err := os.Stat(genesisPath); errors.Is(err, os.ErrNotExist) {
		// Generate Allocs if it doesn't exist
		if _, err := os.Stat(allocsPath); errors.Is(err, os.ErrNotExist) {
			stateDump, err := allocs.GenerateAllocs(monorepo, endpoint)
			if err != nil {
				return err
			}
			utils.WriteJson(allocsPath, stateDump)
		}

		if err := genesis.Generate(monorepo, endpoint); err != nil {
			return err
		}
	}

	// Start L1
	if err := devnet.StartL1(monorepo, endpoint); err != nil {
		return err
	}

	// Generate L2 genesis if it doesn't exist
	if _, err := os.Stat(genesisL2Path); errors.Is(err, os.ErrNotExist) {
		if err := genesis.GenerateL2(monorepo, endpoint); err != nil {
			return err
		}
	}

	// Start L2
	if err := devnet.StartL2(monorepo, l2Endpoint); err != nil {
		return err
	}

	// Bring up the other optimism services
	if err := devnet.StartOpServices(monorepo); err != nil {
		return err
	}

	// Restore the genesis backup
	return genesis.RestoreBackup(monorepo)
}
