package tasks

import (
	"fmt"
	"runtime/debug"

	"golang.org/x/sync/errgroup"
)

// Group is a tasks group, which can at any point be awaited to complete.
// Tasks in the group are run in separate go routines.
// If a task panics, the panic is recovered with HandleCrit.
type Group struct {
	errGroup   errgroup.Group
	HandleCrit func(err error)
}

func (t *Group) Go(fn func() error) {
	t.errGroup.Go(func() error {
		defer func() {
			if err := recover(); err != nil {
				debug.PrintStack()
				t.HandleCrit(fmt.Errorf("panic: %v", err))
			}
		}()
		return fn()
	})
}

func (t *Group) Wait() error {
	return t.errGroup.Wait()
}
