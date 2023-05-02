package main

import (
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
		cmd.RunStepsCommand,
		cmd.GenProofCommand,
	}
}
