package main

import (
	"errors"
	"maps"
	"sort"

	"github.com/ethereum/go-ethereum/log"
	"github.com/urfave/cli/v2"
)

var mergeCommand = &cli.Command{
	Name:   "merge",
	Usage:  "Merge one or more output files into a single file. Later files take precedence per key",
	Flags:  []cli.Flag{FilesFlag, OutputFlag, InputFormatFlag, OutputFormatFlag},
	Action: merge,
}

// merge merges one or more files into a single file
func merge(ctx *cli.Context) error {
	log := log.New()
	files := ctx.StringSlice("files")
	if len(files) < 2 {
		return errors.New("need at least two files to merge")
	}

	log.Info("merging", "files", files)
	reader, ok := formats[ctx.String("input-format")]
	if !ok {
		log.Error("Invalid Input Format. Defaulting to JSON", "Format", ctx.String("input-format"))
		reader = formats["json"]
	}
	writer, ok := formats[ctx.String("output-format")]
	if !ok {
		log.Error("Invalid Output Format. Defaulting to JSON", "Format", ctx.String("output-format"))
		writer = formats["json"]
	}

	aggregates := []aggregate{}
	for _, f := range files {
		a, err := reader.readAggregate(f)
		if err != nil {
			log.Error("failed to read aggregate", "file", f, "err", err)
			return err
		}
		aggregates = append(aggregates, a)
	}

	// sort the aggregates by first block
	sort.Sort(ByFirst(aggregates))

	// check that the block ranges don't have a gap
	err := checkBlockRanges(aggregates)
	if err != nil {
		log.Error("error evaluating block ranges", "err", err)
		return err
	}

	// merge the aggregates
	merged := aggregates[0]
	log.Info("aggregates info", "aggs", aggregates, "len", len(aggregates))
	for _, a := range aggregates[1:] {
		merged = mergeAggregates(merged, a, log)
	}

	// write the merged aggregate
	err = writer.writeAggregate(merged, ctx.String("output"))
	if err != nil {
		log.Error("failed to write aggregate", "err", err)
		return err
	}

	return nil
}

type ByFirst []aggregate

func (a ByFirst) Len() int           { return len(a) }
func (a ByFirst) Less(i, j int) bool { return a[i].First < a[j].First }
func (a ByFirst) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }

// checkBlockRanges checks that the block ranges don't have a gap
// this function assumes the aggregates are sorted by first block
func checkBlockRanges(aggregates []aggregate) error {
	last := aggregates[0].Last
	for _, a := range aggregates[1:] {
		if a.First > last+1 {
			return errors.New("gap in block ranges")
		}
		last = a.Last
	}
	return nil
}

// mergeAggregates merges two aggregates
// this function assumes the aggregates are sorted by first block
func mergeAggregates(a1, a2 aggregate, log log.Logger) aggregate {
	log.Info("merging", "a1", a1, "a2", a2)
	// merge the results
	maps.Copy(a1.Results, a2.Results)

	a1.Last = a2.Last
	log.Info("result", "aggregate", a1)
	return a1
}
