package cmd

import "github.com/urfave/cli/v2"

func RunSteps(ctx *cli.Context) error {
	// TODO
	return nil
}

var RunStepsCommand = &cli.Command{
	Name:        "run-steps",
	Usage:       "",
	Description: "",
	Action:      RunSteps,
	Flags:       nil,
}
