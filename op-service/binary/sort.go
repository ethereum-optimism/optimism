package binary

// SearchWithError is like [sort.Search] but allows f to return an error, and exits earlier on error.
// The int value returned is the same as what sort.Search returns if no error.
func SearchWithError(n int, f func(int) (bool, error)) (int, error) {
	// Define f(-1) == false and f(n) == true.
	// Invariant: f(i-1) == false, f(j) == true.
	i, j := 0, n
	for i < j {
		h := int(uint(i+j) >> 1) // avoid overflow when computing h
		// i ≤ h < j
		ok, err := f(h)
		if err != nil {
			return -1, err
		}
		if !ok {
			i = h + 1 // preserves f(i-1) == false
		} else {
			j = h // preserves f(j) == true
		}
	}
	// i == j, f(i-1) == false, and f(j) (= f(i)) == true  =>  answer is i.
	return i, nil
}
