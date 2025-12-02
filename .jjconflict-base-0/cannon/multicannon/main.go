package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/ethereum-optimism/optimism/cannon/multicannon/version"
	opservice "github.com/ethereum-optimism/optimism/op-service"
	"github.com/ethereum-optimism/optimism/op-service/ctxinterrupt"
	"github.com/urfave/cli/v2"
)

var (
	GitCommit = ""
	GitDate   = ""
)

// VersionWithMeta holds the textual version string including the metadata.
var VersionWithMeta = opservice.FormatVersion(version.Version, GitCommit, GitDate, version.Meta)

func main() {
	app := cli.NewApp()
	app.Name = "multicannon"
	app.Usage = "MIPS Fault Proof tool"
	app.Description = "MIPS Fault Proof tool"
	app.Version = VersionWithMeta
	app.Commands = []*cli.Command{
		LoadELFCommand,
		WitnessCommand,
		RunCommand,
		ListCommand,
	}
	ctx := ctxinterrupt.WithCancelOnInterrupt(context.Background())
	err := app.RunContext(ctx, os.Args)
	if err != nil {
		if errors.Is(err, ctx.Err()) {
			_, _ = fmt.Fprintf(os.Stderr, "command interrupted")
			os.Exit(130)
		} else {
			_, _ = fmt.Fprintf(os.Stderr, "error: %v", err)
			os.Exit(1)
		}
	}
}
