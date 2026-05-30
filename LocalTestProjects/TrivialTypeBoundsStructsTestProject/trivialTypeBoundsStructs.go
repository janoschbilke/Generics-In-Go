package localtestproject

type SimpleGenericWithAny[T any] struct {
	_ T
}

type SimpleGenericWithEmptyInterface[T interface{}] struct {
	_ T
}

type EmptyInterface interface{}

type AnotherEmptyInterface interface{}

type ContainerWithEmptyInterface[T EmptyInterface] struct {
	value T
}

type StorageWithEmptyInterface[T AnotherEmptyInterface] struct {
	items []T
}

func (c *ContainerWithEmptyInterface[T]) Get() T {
	return c.value
}

func (c *ContainerWithEmptyInterface[T]) Set(val T) {
	c.value = val
}

func (s *StorageWithEmptyInterface[T]) Add(item T) {
	s.items = append(s.items, item)
}
