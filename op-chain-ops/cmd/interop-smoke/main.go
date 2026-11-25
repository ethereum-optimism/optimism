package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/urfave/cli/v2"

	"github.com/ethereum-optimism/optimism/op-chain-ops/interopsmoke"
	opservice "github.com/ethereum-optimism/optimism/op-service"
)

var (
	Version     = "v0.0.0"
	VersionMeta = "dev"
	GitCommit   string
	GitDate     string
)

const envPrefix = "INTEROP_SMOKE"

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, os.Interrupt)
	defer cancel()

	app := cli.NewApp()
	app.Name = "interop-smoke"
	app.Usage = "runs interop smoke tests against the RPCs of two live interoperable L2 chains"
	app.Version = opservice.FormatVersion(Version, GitCommit, GitDate, VersionMeta)
	app.Writer = os.Stdout
	app.ErrWriter = os.Stderr
	app.Commands = interopsmoke.Subcommands(envPrefix)

	if err := app.RunContext(ctx, os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
