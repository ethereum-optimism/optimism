package main

import (
	"fmt"
	"github.com/urfave/cli/v2"
	"strconv"
	"strings"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm/versions"
)

func List(ctx *cli.Context) error {
	entries, err := vmFS.ReadDir(baseDir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		filename := entry.Name()
		toks := strings.Split(filename, "-")
		if len(toks) != 2 {
			fmt.Printf("filename: %s\tversion: %s\n", entry.Name(), "unknown")
			continue
		}
		ver, err := strconv.ParseUint(toks[1], 10, 8)
		if err != nil {
			fmt.Printf("filename: %s\tversion: %s\n", entry.Name(), "unknown")
			continue
		}
		fmt.Printf("filename: %s\tversion: %s\n", entry.Name(), versions.StateVersion(ver))
	}
	return nil
}

var ListCommand = &cli.Command{
	Name:        "list",
	Usage:       "List embedded Cannon VM implementations",
	Description: "List embedded Cannon VM implementations",
	Action:      List,
}
