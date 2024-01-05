package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/ethereum-optimism/optimism/op-service/ioutil"
)

func loadJSON[X any](inputPath string) (*X, error) {
	if inputPath == "" {
		return nil, errors.New("no path specified")
	}
	var f io.ReadCloser
	f, err := ioutil.OpenDecompressed(inputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %q: %w", inputPath, err)
	}
	defer f.Close()
	var state X
	if err := json.NewDecoder(f).Decode(&state); err != nil {
		return nil, fmt.Errorf("failed to decode file %q: %w", inputPath, err)
	}
	return &state, nil
}

func writeJSON[X any](outputPath string, value X) error {
	if outputPath == "" {
		return nil
	}
	var out io.Writer
	finish := func() error { return nil }
	if outputPath != "-" {
		f, err := ioutil.NewAtomicWriterCompressed(outputPath, 0755)
		if err != nil {
			return fmt.Errorf("failed to open output file: %w", err)
		}
		// Ensure we close the stream even if failures occur.
		defer f.Close()
		out = f
		// Closing the file causes it to be renamed to the final destination
		// so make sure we handle any errors it returns
		finish = f.Close
	} else {
		out = os.Stdout
	}
	enc := json.NewEncoder(out)
	if err := enc.Encode(value); err != nil {
		return fmt.Errorf("failed to encode to JSON: %w", err)
	}
	_, err := out.Write([]byte{'\n'})
	if err != nil {
		return fmt.Errorf("failed to append new-line: %w", err)
	}
	if err := finish(); err != nil {
		return fmt.Errorf("failed to finish write: %w", err)
	}
	return nil
}
