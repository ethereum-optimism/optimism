package main

import (
	"encoding/gob"
	"fmt"
	"os"
)

// TODO: could put read/writeGob and read/writeJSON into an interface
// and allow for native gob usage during collection. Does not seem worth it at this time.

func writeGob(a aggregate, o string) error {
	if o == "" {
		o = fmt.Sprintf("%d-%d.gob", a.First, a.Last)
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

func readGob(f string) (aggregate, error) {
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
