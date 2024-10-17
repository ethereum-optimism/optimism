package entrydb

import "errors"

var (
	// ErrOutOfOrder happens when you try to add data to the DB,
	// but it does not actually fit onto the latest data (by being too old or new).
	ErrOutOfOrder = errors.New("data out of order")
	// ErrDataCorruption happens when the underlying DB has some I/O issue
	ErrDataCorruption = errors.New("data corruption")
	// ErrSkipped happens when we try to retrieve data that is not available (pruned)
	// It may also happen if we erroneously skip data, that was not considered a conflict, if the DB is corrupted.
	ErrSkipped = errors.New("skipped data")
	// ErrFuture happens when data is just not yet available
	ErrFuture = errors.New("future data")
	// ErrConflict happens when we know for sure that there is different canonical data
	ErrConflict = errors.New("conflicting data")
	// ErrStop can be used in iterators to indicate iteration has to stop
	ErrStop = errors.New("iter stop")
)
