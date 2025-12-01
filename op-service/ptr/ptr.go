// Package ptr provides helper functions for working with pointers.
package ptr

import "fmt"

var (
	Zero16 = New[uint16](0)
	Zero32 = New[uint32](0)
	Zero64 = New[uint64](0)
)

func New[T any](v T) *T {
	return &v
}

func Str[T any](v *T) string {
	if v == nil {
		return "<nil>"
	}
	return fmt.Sprintf("%v", *v)
}
