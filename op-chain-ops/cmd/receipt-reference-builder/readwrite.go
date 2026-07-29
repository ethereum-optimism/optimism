package main

import (
	"encoding/gob"
	"encoding/json"
	"fmt"
	"os"
)

type aggregateReaderWriter interface {
	writeAggregate(a aggregate, o string) error
	readAggregate(f string) (aggregate, error)
}

type jsonAggregateReaderWriter struct{}

// writeAggregate writes the aggregate to a file in json format
// if the output file is not specified, it will create a file based on the block range
func (w jsonAggregateReaderWriter) writeAggregate(a aggregate, o string) error {
	if o == "" {
		o = fmt.Sprintf("%d.%d-%d.json", a.ChainID, a.First, a.Last)
	}
	// write the results to a file
	aggregateJson, err := json.Marshal(a)
	if err != nil {
		return err
	}
	err = os.WriteFile(o, aggregateJson, 0644)
	return err
}

// readAggregate reads the aggregate from a file in json format
func (w jsonAggregateReaderWriter) readAggregate(f string) (aggregate, error) {
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

type gobAggregateReaderWriter struct{}

// writeAggregate writes the aggregate to a file in gob format
// if the output file is not specified, it will create a file based on the block range
func (w gobAggregateReaderWriter) writeAggregate(a aggregate, o string) error {
	if o == "" {
		o = fmt.Sprintf("%d.%d-%d.gob", a.ChainID, a.First, a.Last)
	}
	file, err := os.Create(o)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := gob.NewEncoder(file)
	err = encoder.Encode(&a)
	return err
}

// readAggregate reads the aggregate from a file in gob format
func (w gobAggregateReaderWriter) readAggregate(f string) (aggregate, error) {
	file, err := os.Open(f)
	if err != nil {
		return aggregate{}, err
	}
	defer file.Close()

	a := aggregate{}
	decoder := gob.NewDecoder(file)
	err = decoder.Decode(&a)
	return a, err
}
