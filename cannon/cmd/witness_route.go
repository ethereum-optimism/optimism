//go:build !cannon32 && !cannon64
// +build !cannon32,!cannon64

package cmd

import (
	"github.com/urfave/cli/v2"

	"github.com/ethereum-optimism/optimism/cannon/exec"
)

func Witness(ctx *cli.Context) error {
	arch64 := ctx.Bool(WitnessArch64BitsFlag.Name)

	args := ctx.Args().Slice()
	filter := make([]string, 0, len(args))
	for i := 0 ; i < len(args); i++ {
		if args[i] == "--arch64" {
			i++
		} else {
			filter = append(filter, args[i])
		}
	}
	return exec.ExecuteCannon(filter, !arch64)
}

var WitnessArch64BitsFlag = &cli.BoolFlag{
	Name:     "arch64",
	Usage:    "Indicates whether to use 64-bit Cannon state file",
	Value:    false,
	Required: true,
}

var WitnessCommand = &cli.Command{
	Name:        "witness",
	Usage:       "Convert a Cannon JSON state into a binary witness",
	Description: "Convert a Cannon JSON state into a binary witness. The hash of the witness is written to stdout",
	Action:      Witness,
	Flags: []cli.Flag{
		WitnessArch64BitsFlag,
	},
}
