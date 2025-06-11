package cmd

import (
	"fmt"
	"os"

	factory "github.com/ethereum-optimism/optimism/cannon/mipsevm/versions"
	"github.com/ethereum-optimism/optimism/op-service/ioutil"
	"github.com/ethereum-optimism/optimism/op-service/jsonutil"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
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

type response struct {
	WitnessHash common.Hash   `json:"witnessHash"`
	Witness     hexutil.Bytes `json:"witness"`
	Step        uint64        `json:"step"`
	Exited      bool          `json:"exited"`
	ExitCode    uint8         `json:"exitCode"`
}

func Witness(ctx *cli.Context) error {
	input := ctx.Path(WitnessInputFlag.Name)
	witnessOutput := ctx.Path(WitnessOutputFlag.Name)
	state, err := factory.LoadStateFromFile(input)
	if err != nil {
		return fmt.Errorf("invalid input state (%v): %w", input, err)
	}
	witness, h := state.EncodeWitness()
	if witnessOutput != "" {
		if err := os.WriteFile(witnessOutput, witness, 0755); err != nil {
			return fmt.Errorf("writing output to %v: %w", witnessOutput, err)
		}
	}
	output := response{
		WitnessHash: h,
		Witness:     witness,
		Step:        state.GetStep(),
		Exited:      state.GetExited(),
		ExitCode:    state.GetExitCode(),
	}
	if err := jsonutil.WriteJSON(output, ioutil.ToStdOut()); err != nil {
		return fmt.Errorf("failed to write response: %w", err)
	}
	return nil
}

func CreateWitnessCommand(action cli.ActionFunc) *cli.Command {
	return &cli.Command{
		Name:        "witness",
		Usage:       "Convert a Cannon JSON state into a binary witness",
		Description: "Convert a Cannon JSON state into a binary witness. Basic data about the state is printed to stdout in JSON format.",
		Action:      action,
		Flags: []cli.Flag{
			WitnessInputFlag,
			WitnessOutputFlag,
		},
	}
}

var WitnessCommand = CreateWitnessCommand(Witness)
