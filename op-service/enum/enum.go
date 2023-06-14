package enum

import (
	"fmt"
	"strings"
)

// Stringered wraps the string type to implement the fmt.Stringer interface.
type Stringered string

// String returns the string value.
func (s Stringered) String() string {
	return string(s)
}

// StringeredList converts a list of strings to a list of Stringered.
func StringeredList(values []string) []Stringered {
	var out []Stringered
	for _, v := range values {
		out = append(out, Stringered(v))
	}
	return out
}

// EnumString returns a comma-separated string of the enum values.
// This is primarily used to generate a cli flag.
func EnumString[T fmt.Stringer](values []T) string {
	var out strings.Builder
	for i, v := range values {
		out.WriteString(v.String())
		if i+1 < len(values) {
			out.WriteString(", ")
		}
	}
	return out.String()
}
