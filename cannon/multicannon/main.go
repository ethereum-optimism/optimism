package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/ethereum-optimism/optimism/op-service/ctxinterrupt"
	"github.com/urfave/cli/v2"
)

func main() {
	app := cli.NewApp()
	app.Name = "multicannon"
	app.Usage = "MIPS Fault Proof tool"
	app.Description = "MIPS Fault Proof tool"
	app.Commands = []*cli.Command{
		LoadELFCommand,
		WitnessCommand,
		RunCommand,
	}
	ctx := ctxinterrupt.WithSignalWaiterMain(context.Background())
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
