package localtestproject

// Generic Function Examples

func GenericMax[T int | float64](a, b T) T {
	if a > b {
		return a
	}
	return b
}

func GenericMin[T int | float64](a, b T) T {
	if a < b {
		return a
	}
	return b
}