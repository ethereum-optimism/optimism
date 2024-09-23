package main

import (
	"os"

	"github.com/ethereum-optimism/optimism/cannon/cmd"
	"github.com/urfave/cli/v2"
)

func Witness(ctx *cli.Context) error {
	version, err := parseVersionFlag(ctx.String(WitnessVersionFlag.Name))
	if err != nil {
		return err
	}
	args := removeArg(os.Args[1:], "--version")
	return ExecuteCannon(args, version)
}

var WitnessVersionFlag = &cli.StringFlag{
	Name:     "version",
	Usage:    "Indicates the cannon version to use for witness gen",
	Value:    "",
	Required: true,
}

func createVersionedWitnessCommand() *cli.Command {
	cmd := cmd.CreateWitnessCommand(Witness)
	cmd.Flags = append(cmd.Flags, WitnessVersionFlag)
	return cmd
}

var WitnessCommand = createVersionedWitnessCommand()
