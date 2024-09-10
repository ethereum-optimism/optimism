package jsonutil

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/ethereum-optimism/optimism/op-service/ioutil"
)

func LoadJSON[X any](inputPath string) (*X, error) {
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
	decoder := json.NewDecoder(f)
	if err := decoder.Decode(&state); err != nil {
		return nil, fmt.Errorf("failed to decode file %q: %w", inputPath, err)
	}
	// We are only expecting 1 JSON object - confirm there is no trailing data
	if _, err := decoder.Token(); err != io.EOF {
		return nil, fmt.Errorf("unexpected trailing data in file %q", inputPath)
	}
	return &state, nil
}

func WriteJSON[X any](value X, target ioutil.OutputTarget) error {
	out, closer, abort, err := target()
	if err != nil {
		return err
	}
	if out == nil {
		return nil // No output stream selected so skip generating the content entirely
	}
	defer abort()
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	if err := enc.Encode(value); err != nil {
		return fmt.Errorf("failed to encode to JSON: %w", err)
	}
	_, err = out.Write([]byte{'\n'})
	if err != nil {
		return fmt.Errorf("failed to append new-line: %w", err)
	}
	if err := closer.Close(); err != nil {
		return fmt.Errorf("failed to finish write: %w", err)
	}
	return nil
}
