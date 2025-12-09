package dberrors

import "errors"

var (
	// Out-of-order write or append attempt
	ErrOutOfOrder = errors.New("data out of order")
	// Underlying data corruption or I/O issue
	ErrDataCorruption = errors.New("data corruption")
	// Requested data is pruned or intentionally skipped
	ErrSkipped = errors.New("skipped data")
	// Data not yet available
	ErrFuture = errors.New("future data")
	// Conflicting canonical data detected
	ErrConflict = errors.New("conflicting data")
	// Iterator/control flow stop
	ErrStop = errors.New("iter stop")
	// Previous to first entry (e.g., parent of first block)
	ErrPreviousToFirst = errors.New("cannot get parent of first block in the database")
)
