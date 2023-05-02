package cmd

import "github.com/urfave/cli/v2"

func LoadELF(ctx *cli.Context) error {
	// TODO
	return nil
}

var LoadELFCommand = &cli.Command{
	Name:        "load-elf",
	Usage:       "",
	Description: "",
	Action:      LoadELF,
	Flags:       nil,
}
