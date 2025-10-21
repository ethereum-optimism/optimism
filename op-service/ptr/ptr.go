// Package ptr provides helper functions for working with pointers.
package ptr

import "fmt"

func New[T any](v T) *T {
	return &v
}

func Str[T any](v *T) string {
	if v == nil {
		return "<nil>"
	}
	return fmt.Sprintf("%v", *v)
}
