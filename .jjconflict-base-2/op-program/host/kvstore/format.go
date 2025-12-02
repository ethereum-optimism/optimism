package kvstore

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/ethereum-optimism/optimism/op-program/host/types"
	"github.com/ethereum/go-ethereum/log"
)

const formatFilename = "kvformat"

var (
	ErrFormatUnavailable = errors.New("format unavailable")
	ErrUnsupportedFormat = errors.New("unsupported format")
)

func recordKVFormat(dir string, format types.DataFormat) error {
	return os.WriteFile(filepath.Join(dir, formatFilename), []byte(format), 0o644)
}

func readKVFormat(dir string) (types.DataFormat, error) {
	data, err := os.ReadFile(filepath.Join(dir, formatFilename))
	if errors.Is(err, os.ErrNotExist) {
		return "", ErrFormatUnavailable
	} else if err != nil {
		return "", fmt.Errorf("failed to read kv format: %w", err)
	}
	format := types.DataFormat(data)
	if !slices.Contains(types.SupportedDataFormats, format) {
		return "", fmt.Errorf("%w: %s", ErrUnsupportedFormat, format)
	}
	return format, nil
}

// NewDiskKV creates a new KV implementation. If the specified directly contains an existing KV store
// that has the format recorded, the recorded format is used ensuring compatibility with the existing data.
// If the directory does not contain existing data or doesn't have the format recorded, defaultFormat is used
// which may result in the existing data being unused.
// If the existing data records a format that is not supported, an error is returned.
// The format is automatically recorded if it wasn't previously stored.
func NewDiskKV(logger log.Logger, dir string, defaultFormat types.DataFormat) (KV, error) {
	format, err := readKVFormat(dir)
	if errors.Is(err, ErrFormatUnavailable) {
		format = defaultFormat
		logger.Info("Creating disk storage", "datadir", dir, "format", format)
		if err := recordKVFormat(dir, format); err != nil {
			return nil, fmt.Errorf("failed to record new kv store format: %w", err)
		}
	} else if err != nil {
		return nil, err
	} else {
		logger.Info("Using existing disk storage", "datadir", dir, "format", format)
	}

	switch format {
	case types.DataFormatFile:
		return newFileKV(dir), nil
	case types.DataFormatDirectory:
		return newDirectoryKV(dir), nil
	case types.DataFormatPebble:
		return newPebbleKV(dir), nil
	default:
		return nil, fmt.Errorf("invalid data format: %s", format)
	}
}
