package binary

// SearchL is a binary search variant, which uses an indicator func f(i), and finds the index of the
// last element, and the element itself, which returns true.
//
// Example search space:
// index:  0, 1, 2, 3, 4, 5, 6, 7, 8, 9
// values: 1, 1, 1, 1, 1, 0, 0, 0, 0, 0
//
// SearchL would return: index 4 and f(4)
// Returns -1, if f(i) returns false for all i in [0,n)
// Returns n-1, if f(i) returns true for all i in [0,n)
//
// Based on: https://pesho-ivanov.github.io/#Binary%20search
func SearchL[T any](n int, f func(int) (bool, T, error)) (int, T, error) {
	var zero, elLeft T
	l, r := -1, n
	for r-l > 1 {
		m := int(uint(r+l) >> 1) // avoid overflow when computing m; always in [0,...,n)
		ok, current, err := f(m)
		if err != nil {
			return -1, zero, err
		}
		if ok {
			l = m // l<m => shrinking
			elLeft = current
		} else {
			r = m // r>m => shrinking
		}
	}

	// caller must check `l` for out of bounds
	return l, elLeft, nil
}

// SearchR is the same as SearchL, but returns the index of the first element which returns false.
//
// Example search space:
// index:  0, 1, 2, 3, 4, 5, 6, 7, 8, 9
// values: 1, 1, 1, 1, 1, 0, 0, 0, 0, 0
//
// SearchR would return: index 5 and f(5)
// Returns 0, if f(i) returns false for all i in [0,n)
// Returns n, if f(i) returns true for all i in [0,n)

func SearchR[T any](n int, f func(int) (bool, T, error)) (int, T, error) {
	var zero, elRight T
	l, r := -1, n
	for r-l > 1 {
		m := int(uint(r+l) >> 1) // avoid overflow when computing m; always in [0,...,n)
		ok, current, err := f(m)
		if err != nil {
			return -1, zero, err
		}
		if ok {
			l = m // l<m => shrinking
		} else {
			r = m // r>m => shrinking
			elRight = current
		}
	}

	// caller must check `r` for out of bounds
	return r, elRight, nil
}
