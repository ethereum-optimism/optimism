package entrydb

import (
	"errors"
	"fmt"
	"io"
)

type Iterator[T EntryType, K IndexKey, S IndexState[T, K]] interface {
	TraverseConditional(fn func(state S) error) error
	State() S
}

type iterator[T EntryType, K IndexKey, S IndexState[T, K], D IndexDriver[T, K, S]] struct {
	db          *DB[T, K, S, D]
	current     S
	entriesRead int64
}

func (i *iterator[T, K, S, D]) State() S {
	return i.current
}

// End traverses the iterator to the end of the DB.
// It does not return io.EOF or ErrFuture.
func (i *iterator[T, K, S, D]) End() error {
	for {
		err := i.next()
		if errors.Is(err, ErrFuture) {
			return nil
		} else if err != nil {
			return err
		}
	}
}

func (i *iterator[T, K, S, D]) TraverseConditional(fn func(state S) error) error {
	snapshot := i.db.driver.NewState(0)
	for {
		i.db.driver.Copy(i.current, snapshot) // copy the iterator state, without allocating a new snapshot each iteration
		err := i.next()
		if err != nil {
			i.current = snapshot
			return err
		}
		if i.current.Incomplete() { // skip intermediate states
			continue
		}
		if err := fn(i.current); err != nil {
			i.current = snapshot
			return err
		}
	}
}

// Read and apply the next entry.
func (i *iterator[T, K, S, D]) next() error {
	index := i.current.NextIndex()
	entry, err := i.db.store.Read(index)
	if err != nil {
		if errors.Is(err, io.EOF) {
			return ErrFuture
		}
		return fmt.Errorf("failed to read entry %d: %w", index, err)
	}
	if err := i.current.ApplyEntry(entry); err != nil {
		return fmt.Errorf("failed to process entry %d to iterator state: %w", index, err)
	}

	i.entriesRead++
	return nil
}

func (i *iterator[T, K, S, D]) NextIndex() EntryIdx {
	return i.current.NextIndex()
}
