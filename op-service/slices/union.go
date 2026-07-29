package slices

func Union[T comparable](a, b []T) []T {
	seen := make(map[T]struct{})
	var result []T

	for _, v := range a {
		if _, ok := seen[v]; !ok {
			seen[v] = struct{}{}
			result = append(result, v)
		}
	}

	for _, v := range b {
		if _, ok := seen[v]; !ok {
			seen[v] = struct{}{}
			result = append(result, v)
		}
	}

	return result
}
