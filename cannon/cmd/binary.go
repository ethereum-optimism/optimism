package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path"

	"github.com/ethereum-optimism/optimism/op-service/ioutil"
)

// Serializable defines functionality for a type that may be serialized to raw bytes.
type Serializable interface {
	// Serialize encodes the type as raw bytes.
	Serialize(out io.Writer) error

	// Deserialize decodes raw bytes into the type.
	Deserialize(in io.Reader) error
}

func loadSerializedBinary(inputPath string, obj Serializable) error {
	if inputPath == "" {
		return errors.New("no path specified")
	}
	var f io.ReadCloser
	f, err := ioutil.OpenDecompressed(inputPath)
	if err != nil {
		return fmt.Errorf("failed to open file %q: %w", inputPath, err)
	}
	defer f.Close()
	err = obj.Deserialize(f)
	if err != nil {
		return err
	}
	return nil
}

func writeSerializedBinary(outputPath string, value Serializable) error {
	if outputPath == "" {
		return nil
	}
	var out io.Writer
	finish := func() error { return nil }
	if outputPath != "-" {
		// Write to a tmp file but reserve the file extension if present
		tmpPath := outputPath + "-tmp" + path.Ext(outputPath)
		f, err := ioutil.OpenCompressed(tmpPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
		if err != nil {
			return fmt.Errorf("failed to open output file: %w", err)
		}
		defer f.Close()
		out = f
		finish = func() error {
			// Rename the file into place as atomically as the OS will allow
			return os.Rename(tmpPath, outputPath)
		}
	} else {
		out = os.Stdout
	}
	err := value.Serialize(out)
	if err != nil {
		return fmt.Errorf("failed to write binary: %w", err)
	}
	if err := finish(); err != nil {
		return fmt.Errorf("failed to finish write: %w", err)
	}
	return nil
}
