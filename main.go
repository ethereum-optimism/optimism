package main

import (
	"fmt"
	"os"

	"github.com/urfave/cli/v2"

	"cannon/cmd"
)

func main() {
	app := cli.NewApp()
	app.Name = "cannon"
	app.Usage = "MIPS Fault Proof tool"
	app.Description = "MIPS Fault Proof tool"
	app.Commands = []*cli.Command{
		cmd.LoadELFCommand,
		cmd.RunCommand,
	}
	err := app.Run(os.Args)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "error: %v", err)
		os.Exit(1)
	}
}
