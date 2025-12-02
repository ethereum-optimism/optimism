package main

import (
	"github.com/ethereum/go-ethereum/log"
	"github.com/urfave/cli/v2"
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
