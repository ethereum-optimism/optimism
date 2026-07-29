package tasks

import "context"

// Await waits for a value, and sets it to the destination value.
// This returns an error if the context closes before a value is received from the channel.
func Await[E any](ctx context.Context, src chan E, dest *E) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case x := <-src:
		*dest = x
		return nil
	}
}
