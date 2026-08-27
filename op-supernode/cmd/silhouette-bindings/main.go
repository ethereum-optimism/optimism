package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/urfave/cli/v2"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-core/interop/depset"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-supernode/silhouette"
)

type output struct {
	RollupConfigHash common.Hash `json:"rollupConfigHash"`
	DepSetHash       common.Hash `json:"depSetHash"`
}

func main() {
	app := &cli.App{
		Name:  "silhouette-bindings",
		Usage: "Compute proof-batch binding hashes from parsed network artifacts",
		Flags: []cli.Flag{
			&cli.PathFlag{Name: "rollup-config", Required: true},
			&cli.PathFlag{Name: "dependency-set", Required: true},
		},
		Action: run,
	}
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func readJSON(path string, dst any) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(raw, dst); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	return nil
}

func run(ctx *cli.Context) error {
	var rollupCfg rollup.Config
	if err := readJSON(ctx.Path("rollup-config"), &rollupCfg); err != nil {
		return err
	}
	var depSet depset.StaticConfigDependencySet
	if err := readJSON(ctx.Path("dependency-set"), &depSet); err != nil {
		return err
	}
	rollupHash, depSetHash, err := silhouette.BindingHashes(&rollupCfg, &depSet)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(output{RollupConfigHash: rollupHash, DepSetHash: depSetHash})
}
