package main

import (
	"github.com/urfave/cli/v2"

	"github.com/ethereum-optimism/optimism/op-service/log"
)

var printCommand = &cli.Command{
	Name:   "print",
	Usage:  "read an aggregate file and print it to stdout",
	Flags:  []cli.Flag{FilesFlag, InputFormatFlag},
	Action: print,
}

func print(ctx *cli.Context) error {
	log := log.New()
	files := ctx.StringSlice("files")
	r := formats[ctx.String("input-format")]
	for _, f := range files {
		a, err := r.readAggregate(f)
		if err != nil {
			log.Error("failed to read aggregate", "file", f, "err", err)
			return err
		}
		log.Info("aggregate", "aggregate", a)
	}
	return nil
}
