package safemath

import "golang.org/x/exp/constraints"

// SaturatingAdd adds two unsigned integer values (of the same type),
// and caps the result at the max value of the type.
func SaturatingAdd[V constraints.Unsigned](a, b V) V {
	out, overflow := SafeAdd(a, b)
	if overflow {
		return ^V(0) // max value
	}
	return out
}

// SafeAdd adds two unsigned integer values (of the same type),
// and allows integer overflows, and returns if it overflowed.
func SafeAdd[V constraints.Unsigned](a, b V) (out V, overflow bool) {
	out = a + b
	overflow = out < a
	return
}

// SaturatingSub subtracts two unsigned integer values (of the same type),
// and floors the result at zero.
func SaturatingSub[V constraints.Unsigned](a, b V) V {
	out, underflow := SafeSub(a, b)
	if underflow {
		return V(0) // min value
	}
	return out
}

// SafeSub subtracts two unsigned integer values (of the same type),
// and allows integer underflows, and returns if it underflowed.
func SafeSub[V constraints.Unsigned](a, b V) (out V, underflow bool) {
	out = a - b
	underflow = out > a
	return
}
