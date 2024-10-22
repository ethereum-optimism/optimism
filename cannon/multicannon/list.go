package main

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/urfave/cli/v2"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm/versions"
)

func List(ctx *cli.Context) error {
	return list()
}

func list() error {
	fmt.Println("Available cannon versions:")
	artifacts, err := getArtifacts()
	if err != nil {
		return err
	}
	for _, art := range artifacts {
		if art.isValid() {
			fmt.Printf("filename: %s\tversion: %s (%d)\n", art.filename, versions.StateVersion(art.ver), art.ver)
		} else {
			fmt.Printf("filename: %s\tversion: %s\n", art.filename, "unknown")
		}
	}
	return nil
}

func getArtifacts() ([]artifact, error) {
	var ret []artifact
	entries, err := vmFS.ReadDir(baseDir)
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		filename := entry.Name()
		toks := strings.Split(filename, "-")
		if len(toks) != 2 {
			continue
		}
		if toks[0] != "cannon" {
			continue
		}
		ver, err := strconv.ParseUint(toks[1], 10, 8)
		if err != nil {
			ret = append(ret, artifact{filename, math.MaxUint64})
			continue
		}
		ret = append(ret, artifact{filename, ver})
	}
	return ret, nil
}

type artifact struct {
	filename string
	ver      uint64
}

func (a artifact) isValid() bool {
	return a.ver != math.MaxUint64
}

var ListCommand = &cli.Command{
	Name:        "list",
	Usage:       "List embedded Cannon VM implementations",
	Description: "List embedded Cannon VM implementations",
	Action:      List,
}
