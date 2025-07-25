// Package main reproduces a lightweight version of the "kurtosis enclave inspect" command
// It can be used to sanity check the results, as writing tests against a fake
// enclave is not practical right now.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/urfave/cli/v2"

	"github.com/ethereum-optimism/optimism/kurtosis-devnet/pkg/kurtosis/sources/inspect"
	"github.com/ethereum-optimism/optimism/kurtosis-devnet/pkg/kurtosis/sources/inspect/flags"
	opservice "github.com/ethereum-optimism/optimism/op-service"
	"github.com/ethereum-optimism/optimism/op-service/cliapp"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
)

var (
	Version   = "v0.1.0"
	GitCommit = ""
	GitDate   = ""
)

func main() {
	app := cli.NewApp()
	app.Version = opservice.FormatVersion(Version, GitCommit, GitDate, "")
	app.Name = "kurtosis-inspect"
	app.Usage = "Inspect Kurtosis enclaves and extract configurations"
	app.Description = "Tool to inspect running Kurtosis enclaves and extract conductor configurations and environment data"
	app.Flags = cliapp.ProtectFlags(flags.Flags)
	app.Action = cliapp.LifecycleCmd(run)
	app.ArgsUsage = "<enclave-id>"

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run(cliCtx *cli.Context, closeApp context.CancelCauseFunc) (cliapp.Lifecycle, error) {
	// Parse configuration
	cfg, err := inspect.NewConfig(cliCtx)
	if err != nil {
		return nil, err
	}

	// Setup logging
	log := oplog.NewLogger(oplog.AppOut(cliCtx), oplog.ReadCLIConfig(cliCtx))
	oplog.SetGlobalLogHandler(log.Handler())

	// Create service
	service := inspect.NewInspectService(cfg, log)

	// Create background context for operations
	ctx := context.Background()

	// Run the service
	if err := service.Run(ctx); err != nil {
		return nil, err
	}

	return nil, nil
}
