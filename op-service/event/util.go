package event

type Matcher func(ev Event) bool

func IsNot[T Event](ev Event) bool {
	_, ok := ev.(T)
	return !ok
}

// Is as helper function is syntax-sugar to do an Event type check as a boolean function
func Is[T Event](ev Event) bool {
	_, ok := ev.(T)
	return ok
}

// Any as helper function combines different event conditions into a single OR expression.
// This returns false if none of the inputs match, or if there are no conditions to check.
func Any(fns ...Matcher) Matcher {
	return func(ev Event) bool {
		for _, fn := range fns {
			if fn(ev) {
				return true
			}
		}
		return false
	}
}

// All as helper function combines different event conditions into a single AND expression.
// This returns true if all functions match, or if there are no conditions to check.
func All(fns ...Matcher) Matcher {
	return func(ev Event) bool {
		for _, fn := range fns {
			if !fn(ev) {
				return false
			}
		}
		return true
	}
}
