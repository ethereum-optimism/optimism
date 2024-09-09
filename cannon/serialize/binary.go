package serialize

import (
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"

	"github.com/ethereum-optimism/optimism/op-service/ioutil"
)

// Serializable defines functionality for a type that may be serialized to raw bytes.
type Serializable interface {
	// Serialize encodes the type as raw bytes.
	Serialize(out io.Writer) error

	// Deserialize decodes raw bytes into the type.
	Deserialize(in io.Reader) error
}

func LoadSerializedBinary[X any](inputPath string) (*X, error) {
	if inputPath == "" {
		return nil, errors.New("no path specified")
	}
	var f io.ReadCloser
	f, err := ioutil.OpenDecompressed(inputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %q: %w", inputPath, err)
	}
	defer f.Close()

	var x X
	serializable, ok := reflect.ValueOf(&x).Interface().(Serializable)
	if !ok {
		return nil, fmt.Errorf("%T is not a Serializable", x)
	}
	err = serializable.Deserialize(f)
	if err != nil {
		return nil, err
	}
	return &x, nil
}

func WriteSerializedBinary(outputPath string, value Serializable, perm os.FileMode) error {
	if outputPath == "" {
		return nil
	}
	var out io.Writer
	finish := func() error { return nil }
	if outputPath == "-" {
		out = os.Stdout
	} else {
		f, err := ioutil.NewAtomicWriterCompressed(outputPath, perm)
		if err != nil {
			return fmt.Errorf("failed to create temp file when writing: %w", err)
		}
		// Ensure we close the stream without renaming even if failures occur.
		defer func() {
			_ = f.Abort()
		}()
		out = f
		// Closing the file causes it to be renamed to the final destination
		// so make sure we handle any errors it returns
		finish = f.Close
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
