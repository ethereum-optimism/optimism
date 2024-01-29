package main

import (
	"errors"

	"github.com/ethereum/go-ethereum/log"
	"github.com/urfave/cli/v2"
)

var printGobCommand = &cli.Command{
	Name:   "print-gob",
	Usage:  "read a gob binary aggregate and print it to screen (for debugging)",
	Flags:  []cli.Flag{FilesFlag, OutputFlag},
	Action: printGob,
}

func printGob(ctx *cli.Context) error {
	log := log.New()
	files := ctx.StringSlice("files")
	if len(files) != 1 {
		return errors.New("only one file is supported")
	}
	for _, f := range files {
		a, err := readGob(f)
		if err != nil {
			log.Error("failed to read aggregate", "file", f, "err", err)
			return err
		}
		log.Info("aggregate", "aggregate", a)
	}
	return nil
}
