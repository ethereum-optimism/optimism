package plasma

import (
	"context"
	"errors"
)

type DataClient interface {
	GetInput(ctx context.Context, key []byte) ([]byte, error)
	SetInput(ctx context.Context, img []byte) ([]byte, error)
}

// ErrNotFound is returned when the server could not find the input.
var ErrNotFound = errors.New("not found")

// ErrCommitmentMismatch is returned when the server returns the wrong input for the given commitment.
var ErrCommitmentMismatch = errors.New("commitment mismatch")

// ErrInvalidInput is returned when the input is not valid for posting to the DA storage.
var ErrInvalidInput = errors.New("invalid input")
