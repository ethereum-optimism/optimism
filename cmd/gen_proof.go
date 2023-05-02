package cmd

import "github.com/urfave/cli/v2"

func GenProof(ctx *cli.Context) error {
	// TODO
	return nil
}

var GenProofCommand = &cli.Command{
	Name:        "gen-proof",
	Usage:       "",
	Description: "",
	Action:      GenProof,
	Flags:       nil,
}
