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
// This variant can easily be updated to return the first element, which returns false if necessary, (i.e. the SearchR variant)
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

	if l == -1 { // all false
		return -1, zero, nil
	}

	return l, elLeft, nil
}
