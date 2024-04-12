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

func Flatten[T any](a [][]T) []T {
	res := []T{}

	for _, sl := range a {
		for _, el := range sl {
			res = append(res, el)
		}
	}

	return res
}
