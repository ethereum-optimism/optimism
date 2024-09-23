package main

import (
	"os"

	"github.com/ethereum-optimism/optimism/cannon/cmd"
	"github.com/urfave/cli/v2"
)

func LoadELF(ctx *cli.Context) error {
	ver, err := parseVersionFlag(ctx.String(LoadELFVersionFlag.Name))
	if err != nil {
		return err
	}
	args := removeArg(os.Args[1:], "--version")
	return ExecuteCannon(args, ver)
}

func createVersionedLoadELFCommand() *cli.Command {
	cmd := cmd.CreateLoadELFCommand(LoadELF)
	cmd.Flags = append(cmd.Flags, LoadELFVersionFlag)
	return cmd
}

var LoadELFVersionFlag = &cli.StringFlag{
	Name:     "version",
	Usage:    "Indicates the cannon version to use for state loading",
	Value:    "",
	Required: true,
}

var LoadELFCommand = createVersionedLoadELFCommand()
