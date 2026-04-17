package diff

// WalkLineRanges invokes fn for every line number inside a
// line_ranges value. Handles both raw [][2]int (from in-process
// graph state) and []interface{} (from JSON round-tripped graph).
// Unknown shapes are silently skipped.
//
// Single consolidated walker used by both coverage-intersection
// signal math and per-hunk freshness diffing, so the shape-handling
// logic lives in one place.
func WalkLineRanges(ranges interface{}, fn func(lineNum int)) {
	switch rs := ranges.(type) {
	case [][2]int:
		for _, r := range rs {
			for l := r[0]; l <= r[1]; l++ {
				fn(l)
			}
		}
	case []interface{}:
		for _, r := range rs {
			pair, ok := r.([]interface{})
			if !ok || len(pair) != 2 {
				continue
			}
			start, ok1 := asInt(pair[0])
			end, ok2 := asInt(pair[1])
			if !ok1 || !ok2 {
				continue
			}
			for l := start; l <= end; l++ {
				fn(l)
			}
		}
	}
}

// asInt coerces a JSON-numeric-shaped value to int. Returns
// (0, false) for non-numeric inputs.
func asInt(v interface{}) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, true
	case int64:
		return int(n), true
	case float64:
		return int(n), true
	}
	return 0, false
}
