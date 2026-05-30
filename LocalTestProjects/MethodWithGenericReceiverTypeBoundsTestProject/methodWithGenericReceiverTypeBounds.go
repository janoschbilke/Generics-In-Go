package localtestproject

type I[T any] interface {
	m(T)
}

type G[T any] struct {
	_ T
}

type G2[T I[T]] struct {
	_ T
}

func (x G[T]) someMethod()     {}
func (x G[T]) anotherMethod()  {}
func (x G2[T]) someMethod()    {}
func (x G2[T]) anotherMethod() {}

type SimpleContainer[T interface{}] struct {
	item T
}

func (c SimpleContainer[T]) Get() T {
	return c.item
}

type ComparableContainer[T comparable] struct {
	items []T
}

func (c ComparableContainer[T]) Contains(item T) bool {
	for _, v := range c.items {
		if v == item {
			return true
		}
	}
	return false
}
