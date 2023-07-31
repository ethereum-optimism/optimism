package cmd

import (
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

func loadJSON[X any](inputPath string) (*X, error) {
	if inputPath == "" {
		return nil, errors.New("no path specified")
	}
	var f io.ReadCloser
	f, err := os.OpenFile(inputPath, os.O_RDONLY, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %q: %w", inputPath, err)
	}
	defer f.Close()
	if isGzip(inputPath) {
		f, err = gzip.NewReader(f)
		if err != nil {
			return nil, fmt.Errorf("create gzip reader: %w", err)
		}
		defer f.Close()
	}
	var state X
	if err := json.NewDecoder(f).Decode(&state); err != nil {
		return nil, fmt.Errorf("failed to decode file %q: %w", inputPath, err)
	}
	return &state, nil
}

func writeJSON[X any](outputPath string, value X, outIfEmpty bool) error {
	var out io.Writer
	if outputPath != "" {
		f, err := os.OpenFile(outputPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
		if err != nil {
			return fmt.Errorf("failed to open output file: %w", err)
		}
		defer f.Close()
		out = f
		if isGzip(outputPath) {
			g := gzip.NewWriter(f)
			defer g.Close()
			out = g
		}
	} else if outIfEmpty {
		out = os.Stdout
	} else {
		return nil
	}
	enc := json.NewEncoder(out)
	if err := enc.Encode(value); err != nil {
		return fmt.Errorf("failed to encode to JSON: %w", err)
	}
	_, err := out.Write([]byte{'\n'})
	if err != nil {
		return fmt.Errorf("failed to append new-line: %w", err)
	}
	return nil
}

func isGzip(path string) bool {
	return strings.HasSuffix(path, ".gz")
}
