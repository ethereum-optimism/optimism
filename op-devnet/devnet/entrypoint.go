package devnet

import (
	"errors"
	"os"

	"github.com/ethereum/go-ethereum/log"
	"github.com/urfave/cli/v2"

	"github.com/ethereum-optimism/optimism/op-devnet/flags"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/opio"
)

// Main is the entrypoint for the devnet.
func Main(cliCtx *cli.Context) error {
	log.Info("Initializing devnet")

	// Use the current directory as the monorepo directory if not specified.
	monorepoDir := cliCtx.String(flags.MonorepoDir.Name)
	if monorepoDir == "" {
		monoDir, err := os.Getwd()
		if err != nil {
			log.Crit("Failed to get current working directory", "err", err)
		}
		monorepoDir = monoDir
	}

	// Read the deploy flag.
	deploy := cliCtx.Bool(flags.Deploy.Name)
	if deploy {
		log.Info("Deploying contracts")
	}

	// Block on interrupts.
	opio.BlockOnInterrupts()

	return nil
}



