package binary

// SearchFirst is a copy-paste from sort.Search, but includes memoization and stops on and returns error
// Returns the first index when f(i) within [0,n) return true
// Returns n if f(i) returns false for all i in [0,n)
// Returns 0 if f(i) returns true for all i in [0,n)
func SearchFirst[T any](n int, f func(int) (bool, T, error)) (int, T, error) {
	mem := make(map[int]T)
	var zero T
	// Define f(-1) == false and f(n) == true.
	// Invariant: f(i-1) == false, f(j) == true.
	i, j := 0, n
	for i < j {
		h := int(uint(i+j) >> 1) // avoid overflow when computing h
		// i ≤ h < j
		ok, current, err := f(h)
		if err != nil {
			return -1, zero, err
		}
		mem[h] = current
		if !ok {
			i = h + 1 // preserves f(i-1) == false
		} else {
			j = h // preserves f(j) == true
		}
	}
	// i == j, f(i-1) == false, and f(j) (= f(i)) == true  =>  answer is i.
	return i, mem[i], nil
}

// SearchL is based on https://pesho-ivanov.github.io/#Binary%20search, includes memoization and returns the element on the left of the border
// Returns -1, if f(i) returns false for all i in [0,n)
// Returns n, if f(i) returns true for all i in [0,n)
func SearchL[T any](n int, f func(int) (bool, T, error)) (int, T, error) {
	mem := make(map[int]T)
	var zero T
	l, r := -1, n
	for r-l > 1 {
		m := int(uint(r+l) >> 1) // avoid overflow when computing m; always in [0,...,n)
		ok, current, err := f(m)
		if err != nil {
			return -1, zero, err
		}
		mem[m] = current
		if ok {
			l = m // l<m => shrinking
		} else {
			r = m // r>m => shrinking
		}
	}

	if r == n { // all true
		return n, zero, nil
	}
	if l == -1 { // all false
		return -1, zero, nil
	}

	return l, mem[l], nil
}
