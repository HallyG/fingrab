package sliceutil

func Filter[T any](list []T, filter func(T) bool) []T {
	filtered := make([]T, 0)

	for _, element := range list {
		if filter(element) {
			filtered = append(filtered, element)
		}
	}

	return filtered
}
