package main

import (
	"os"

	"github.com/ethereum-optimism/optimism/cannon/cmd"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/versions"
	"github.com/urfave/cli/v2"
)

func LoadELF(ctx *cli.Context) error {
	typ, err := parseFlag(os.Args[1:], "--type")
	if err != nil {
		return err
	}
	ver, err := versions.ParseStateVersion(typ)
	if err != nil {
		return err
	}
	return ExecuteCannon(ctx.Context, os.Args[1:], ver)
}

var LoadELFCommand = cmd.CreateLoadELFCommand(LoadELF)
