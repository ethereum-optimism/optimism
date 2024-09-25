package main

import (
	"fmt"
	"os"

	"github.com/urfave/cli/v2"

	"github.com/ethereum-optimism/optimism/cannon/cmd"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/versions"
)

func Witness(ctx *cli.Context) error {
	if len(os.Args) == 3 && os.Args[2] == "--help" {
		if err := list(); err != nil {
			return err
		}
		fmt.Println("use `--input <valid input file> --help` to get more detailed help")
	}

	inputPath, err := parsePathFlag(os.Args[1:], "--input")
	if err != nil {
		return err
	}
	version, err := versions.DetectVersion(inputPath)
	if err != nil {
		return err
	}
	return ExecuteCannon(ctx.Context, os.Args[1:], version)
}

var WitnessCommand = cmd.CreateWitnessCommand(Witness)
