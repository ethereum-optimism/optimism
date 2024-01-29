package main

import (
	"errors"

	"github.com/ethereum/go-ethereum/log"
	"github.com/urfave/cli/v2"
)

var generateCommand = &cli.Command{
	Name:   "generate",
	Usage:  "Turn a JSON aggregate into a gob binary aggregate",
	Flags:  []cli.Flag{FilesFlag, OutputFlag},
	Action: toGob,
}

func toGob(ctx *cli.Context) error {
	log := log.New()
	files := ctx.StringSlice("files")
	if len(files) != 1 {
		return errors.New("only one file is supported")
	}
	for _, f := range files {
		a, err := readJSON(f)
		if err != nil {
			log.Error("failed to read aggregate", "file", f, "err", err)
			return err
		}
		writeGob(a, ctx.String("output"))
	}
	return nil
}
