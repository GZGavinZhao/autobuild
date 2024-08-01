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

func Uniq[T comparable](a []T) []T {
	if len(a) == 0 {
		return a
	}

	b := a[:1]
	for i, x := range a {
		if i == 0 {
			continue
		}

		if b[len(b)-1] != x {
			b = append(b, x)
		}
	}
	return b
}

func Uniq2[T comparable](a []T) []T {
	if len(a) == 0 {
		return a 
	}
	l := 0

	for i := range a {
		if i == 0 {
			continue
		}

		if a[i] != a[l] {
			l++
			a[l] = a[i]
		}
	}

	return a[:l+1]
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
