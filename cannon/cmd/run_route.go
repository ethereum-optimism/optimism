//go:build !cannon32 && !cannon64
// +build !cannon32,!cannon64

package cmd

import (
	"github.com/urfave/cli/v2"

	"github.com/ethereum-optimism/optimism/cannon/exec"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/versions"
)

func Run(ctx *cli.Context) error {
	inputPath := ctx.Path(RunInputFlag.Name)
	version, err := versions.DetectVersion(inputPath)
	if err != nil {
		return err
	}
	arch64 := version == versions.VersionMultiThreaded64
	return exec.ExecuteCannon(ctx.Args().Slice(), arch64)
}

var RunCommand = CreateRunCommand(Run)
