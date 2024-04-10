package cmd

import (
	"fmt"
	"os"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm"
	"github.com/ethereum-optimism/optimism/op-service/jsonutil"
	"github.com/urfave/cli/v2"
)

var (
	// WitnessInputFlag is a CLI flag that specifies the path to the input JSON state.
	WitnessInputFlag = &cli.PathFlag{
		Name:     "input",
		Usage:    "path of input JSON state",
		TakesFile: true,
		Required: true,
	}

	// WitnessOutputFlag is a CLI flag that specifies the path to write the binary witness.
	WitnessOutputFlag = &cli.PathFlag{
		Name:     "output",
		Usage:    "path to write binary witness",
		TakesFile: true,
	}
)

// generateWitness is a function that converts a Cannon JSON state into a binary witness.
// It takes the input and output paths from the CLI flags, loads the JSON state, encodes the witness,
// computes the state hash, and writes the binary witness to the output file (if specified).
// The hash of the witness is printed to stdout.
func generateWitness(ctx *cli.Context) error {
	input := ctx.Path(WitnessInputFlag.Name)
	output := ctx.Path(WitnessOutputFlag.Name)

	state, err := jsonutil.LoadJSON[mipsevm.State](input)
	if err != nil {
		return fmt.Errorf("invalid input state (%v): %w", input, err)
	}

	witness := state.EncodeWitness()
	h, err := witness.StateHash()
	if err != nil {
		return fmt.Errorf("failed to compute witness hash: %w", err)
	}

	if output != "" {
		if err := os.WriteFile(output, witness, 0755); err != nil {
			return fmt.Errorf("writing output to %v: %w", output, err)
		}
	}

	fmt.Printf("%x\n", h)
	return nil
}

// WitnessCommand is a CLI command that converts a Cannon JSON state into a binary witness.
// It accepts two flags: "input" (the path to the input JSON state) and "output" (the path to
// write the binary witness). The hash of the witness is written to stdout.
var WitnessCommand = &cli.Command{
	Name:        "witness",
	Usage:       "Convert a Cannon JSON state into a binary witness",
	Description: "Convert a Cannon JSON state into a binary witness. The hash of the witness is written to stdout",
	Action:      generateWitness,
	Flags: []cli.Flag{
		WitnessInputFlag,
		WitnessOutputFlag,
	},
}
