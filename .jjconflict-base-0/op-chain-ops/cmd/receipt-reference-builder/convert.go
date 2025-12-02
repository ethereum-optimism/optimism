package main

import (
	"errors"

	"github.com/ethereum/go-ethereum/log"
	"github.com/urfave/cli/v2"
)

var convertCommand = &cli.Command{
	Name:   "convert",
	Usage:  "convert an aggregate from one format to another",
	Flags:  []cli.Flag{FilesFlag, OutputFlag, InputFormatFlag, OutputFormatFlag},
	Action: convert,
}

func convert(ctx *cli.Context) error {
	log := log.New()
	files := ctx.StringSlice("files")
	if len(files) != 1 {
		return errors.New("only one file is supported")
	}

	if ctx.String("input-format") == ctx.String("output-format") {
		log.Info("no conversion needed. specify different input and output formats")
		return nil
	}

	r := formats[ctx.String("input-format")]
	w := formats[ctx.String("output-format")]

	for _, f := range files {
		a, err := r.readAggregate(f)
		if err != nil {
			log.Error("failed to read aggregate", "file", f, "err", err)
			return err
		}
		err = w.writeAggregate(a, ctx.String("output"))
		if err != nil {
			log.Error("failed to write aggregate", "file", f, "err", err)
			return err
		}

	}
	return nil
}
