package types

import (
	"errors"
)

const maxIDLength = 100

var ErrInvalidID = errors.New("invalid ID")

// SyncTesterID represents a unique syncTester.
type SyncTesterID string

func (id SyncTesterID) String() string {
	return string(id)
}

func (id SyncTesterID) MarshalText() ([]byte, error) {
	if len(id) > maxIDLength {
		return nil, ErrInvalidID
	}
	if len(id) == 0 {
		return nil, ErrInvalidID
	}
	return []byte(id), nil
}

func (id *SyncTesterID) UnmarshalText(data []byte) error {
	if len(data) > maxIDLength {
		return ErrInvalidID
	}
	if len(data) == 0 {
		return ErrInvalidID
	}
	*id = SyncTesterID(data)
	return nil
}

// FCUState represents the Fork Choice Update state with Latest, Safe, and Finalized block numbers
type FCUState struct {
	Latest    uint64
	Safe      uint64
	Finalized uint64
}
