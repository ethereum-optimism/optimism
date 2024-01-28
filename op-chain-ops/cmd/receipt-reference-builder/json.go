package main

import (
	"encoding/json"
	"fmt"
	"os"
)

// writeAggregate writes the aggregate to a file
// if the output file is not specified, it will creeate a file based on the block range
func writeAggregate(a aggregate, o string) error {
	if o == "" {
		o = fmt.Sprintf("%d-%d.json", a.First, a.Last)
	}
	// write the results to a file
	aggregateJson, err := json.Marshal(a)
	if err != nil {
		return err
	}
	os.WriteFile(o, aggregateJson, 0644)
	if err != nil {
		return err
	}
	return nil
}

func readAggregate(f string) (aggregate, error) {
	// read the file
	aggregateJson, err := os.ReadFile(f)
	if err != nil {
		return aggregate{}, err
	}
	var a aggregate
	err = json.Unmarshal(aggregateJson, &a)
	if err != nil {
		return aggregate{}, err
	}
	return a, nil
}
