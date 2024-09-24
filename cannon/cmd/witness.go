package cmd

import (
	"fmt"
	"os"

	factory "github.com/ethereum-optimism/optimism/cannon/mipsevm/versions"
	"github.com/urfave/cli/v2"
)

var (
	WitnessInputFlag = &cli.PathFlag{
		Name:      "input",
		Usage:     "path of input JSON state.",
		TakesFile: true,
		Required:  true,
	}
	WitnessOutputFlag = &cli.PathFlag{
		Name:      "output",
		Usage:     "path to write binary witness.",
		TakesFile: true,
	}
)

func Witness(ctx *cli.Context) error {
	input := ctx.Path(WitnessInputFlag.Name)
	output := ctx.Path(WitnessOutputFlag.Name)
	state, err := factory.LoadStateFromFile(input)
	if err != nil {
		return fmt.Errorf("invalid input state (%v): %w", input, err)
	}
	witness, h := state.EncodeWitness()
	if output != "" {
		if err := os.WriteFile(output, witness, 0755); err != nil {
			return fmt.Errorf("writing output to %v: %w", output, err)
		}
	}
	fmt.Println(h.Hex())
	return nil
}

func CreateWitnessCommand(action cli.ActionFunc) *cli.Command {
	return &cli.Command{
		Name:        "witness",
		Usage:       "Convert a Cannon JSON state into a binary witness",
		Description: "Convert a Cannon JSON state into a binary witness. The hash of the witness is written to stdout",
		Action:      action,
		Flags: []cli.Flag{
			WitnessInputFlag,
			WitnessOutputFlag,
		},
	}
}

var WitnessCommand = CreateWitnessCommand(Witness)
