package main

import (
	"fmt"
	"os"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/cli"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/version"
)

func main() {
	app := cli.NewApp(version.VersionWithMeta)
	app.Writer = os.Stdout
	app.ErrWriter = os.Stderr
	err := app.Run(os.Args)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Application failed: %v\n", err)
		os.Exit(1)
	}
}
