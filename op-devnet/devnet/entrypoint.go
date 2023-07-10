package devnet

import (
	"fmt"
	"os"

	"github.com/ethereum/go-ethereum/log"
	"github.com/urfave/cli/v2"

	"github.com/ethereum-optimism/optimism/op-devnet/flags"
)

// Main is the entrypoint for the devnet.
func Main(cliCtx *cli.Context) error {
	log.Info("Initializing devnet")

	paths, err := NewPaths(cliCtx)
	if err != nil {
		log.Crit("Failed to parse required directories", "err", err)
	}

	err = os.MkdirAll(paths.devnetDir, os.ModePerm)
	if err != nil {
		log.Crit("Failed to create devnet directory", "err", err)
	}

	log.Info("Building ops-bedrock")
	additionalEnv := fmt.Sprintf("PWD=%s", paths.opsBedrockDir)
	if err := RunCommand([]string{"docker-compose", "build", "--progress", "plain"}, []string{additionalEnv}, paths.opsBedrockDir); err != nil {
		log.Crit("Failed to build ops-bedrock", "err", err)
	}

	if cliCtx.Bool(flags.Deploy.Name) {
		DeployContracts(cliCtx, &paths)
	} else {
		Prestate(cliCtx, &paths)
	}

	return nil
}
