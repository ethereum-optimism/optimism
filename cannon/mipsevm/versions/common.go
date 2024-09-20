package versions

import "errors"

type StateVersion uint8

const (
	VersionSingleThreaded StateVersion = iota
	VersionMultiThreaded
	VersionMultiThreaded64
)

var (
	ErrUnknownVersion         = errors.New("unknown version")
	ErrJsonNotSupported       = errors.New("json not supported")
	ErrInvalidStateFileFormat = errors.New("invalid state file format")
	ErrUnsupportedMipsArch    = errors.New("mips architecture is not supported")
)
