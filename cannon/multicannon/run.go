package main

import (
	"os"

	"github.com/urfave/cli/v2"

	"github.com/ethereum-optimism/optimism/cannon/cmd"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/versions"
)

func Run(ctx *cli.Context) error {
	inputPath := ctx.Path(cmd.RunInputFlag.Name)
	version, err := versions.DetectVersion(inputPath)
	if err != nil {
		return err
	}
	return ExecuteCannon(os.Args[1:], version)
}

var RunCommand = cmd.CreateRunCommand(Run)
