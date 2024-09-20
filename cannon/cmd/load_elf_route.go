//go:build !cannon32 && !cannon64
// +build !cannon32,!cannon64

package cmd

import (
	"os"

	"github.com/ethereum-optimism/optimism/cannon/exec"
	"github.com/urfave/cli/v2"
)

func LoadELF(ctx *cli.Context) error {
	arch64 := ctx.Bool(LoadELFArch64BitsFlag.Name)
	args := removeArg(os.Args[1:], "--arch64")
	return exec.ExecuteCannon(args, !arch64)
}

func createMipsxLoadELFCommand() *cli.Command {
	cmd := CreateLoadELFCommand(LoadELF)
	cmd.Flags = append(cmd.Flags, LoadELFArch64BitsFlag)
	return cmd
}

var LoadELFArch64BitsFlag = &cli.BoolFlag{
	Name:     "arch64",
	Usage:    "Indicates whether to load a 64-bit Cannon state file",
	Value:    false,
	Required: true,
}

var LoadELFCommand = createMipsxLoadELFCommand()
