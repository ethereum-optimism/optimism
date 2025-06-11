package safemath

import "golang.org/x/exp/constraints"

// SaturatingAdd adds two unsigned integer values (of the same type), and caps the result at the max value of the type.
func SaturatingAdd[V constraints.Unsigned](a, b V) V {
	sum, overflow := SafeAdd(a, b)
	if overflow {
		return ^V(0) // max value
	}
	return sum
}

// SafeAdd adds two unsigned integer values (of the same type),
// and allows integer overflows, and returns if it overflowed.
func SafeAdd[V constraints.Unsigned](a, b V) (out V, overflow bool) {
	out = a + b
	overflow = out < a
	return
}
