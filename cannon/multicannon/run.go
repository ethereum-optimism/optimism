package main

import (
	"os"

	"github.com/urfave/cli/v2"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm/versions"
)

func Run(ctx *cli.Context) error {
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

// var RunCommand = cmd.CreateRunCommand(Run)
var RunCommand = &cli.Command{
	Name:            "run",
	Usage:           "Run VM step(s) and generate proof data to replicate onchain.",
	Description:     "Run VM step(s) and generate proof data to replicate onchain. See flags to match when to output a proof, a snapshot, or to stop early.",
	Action:          Run,
	SkipFlagParsing: true,
}
