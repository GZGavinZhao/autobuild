package utils

func Filter[T any](a []T, test func(T) bool) []T {
	b := a[:0]

	for _, x := range a {
		if test(x) {
			b = append(b, x)
		}
	}

	return b
}
