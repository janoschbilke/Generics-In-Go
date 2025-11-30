package localtestproject

// TRIVIAL BOUND: Direktes 'any' 
type SimpleGenericWithAny[T any] struct {
	_ T
}

// TRIVIAL BOUND: Direktes leeres Interface 
type SimpleGenericWithEmptyInterface[T interface{}] struct {
	_ T
}

// Erweiterung: Definition von leeren Interfaces muss auch als trivial gelten
type EmptyInterface interface{}

type AnotherEmptyInterface interface{}

// TRIVIAL BOUND: Implizit leeres Interface als Constraint
type ContainerWithEmptyInterface[T EmptyInterface] struct {
	value T
}

// TRIVIAL BOUND: 
type StorageWithEmptyInterface[T AnotherEmptyInterface] struct {
	items []T
}


// Methods with Generic Receiver
func (c *ContainerWithEmptyInterface[T]) Get() T {
	return c.value
}

func (c *ContainerWithEmptyInterface[T]) Set(val T) {
	c.value = val
}

func (s *StorageWithEmptyInterface[T]) Add(item T) {
		s.items = append(s.items, item)
}
