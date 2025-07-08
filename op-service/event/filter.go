package event

import "context"

// Filter is used to filter down events before proceeding to execute a handler Serve method.
// Returns true if the event is accepted.
type Filter func(ctx context.Context, ev Event) bool

// And combines the current filter with the given filter to both be required.
// If f is nil, only the other filter is applied.
// This makes filter functions easy to compose.
func (f Filter) And(other Filter) Filter {
	if f == nil {
		return other
	}
	return func(ctx context.Context, ev Event) bool {
		return f(ctx, ev) && other(ctx, ev)
	}
}
