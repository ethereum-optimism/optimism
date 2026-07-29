package event

// Is as helper function is syntax-sugar to do an Event type check as a boolean function
func Is[T Event](ev Event) bool {
	_, ok := ev.(T)
	return ok
}

// Any as helper function combines different event conditions into a single function
func Any(fns ...func(ev Event) bool) func(ev Event) bool {
	return func(ev Event) bool {
		for _, fn := range fns {
			if fn(ev) {
				return true
			}
		}
		return false
	}
}
