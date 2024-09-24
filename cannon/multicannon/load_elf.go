package main

import (
	"os"

	"github.com/ethereum-optimism/optimism/cannon/cmd"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/versions"
	"github.com/urfave/cli/v2"
)

func LoadELF(ctx *cli.Context) error {
	ver, err := versions.ParseStateVersion(ctx.String(cmd.LoadELFVMTypeFlag.Name))
	if err != nil {
		return err
	}
	return ExecuteCannon(os.Args[1:], ver)
}

var LoadELFCommand = cmd.CreateLoadELFCommand(LoadELF)
