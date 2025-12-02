package localtestproject

// Erweiterung 3: Unterscheidung zwischen trivial und non-trivial Type Bounds bei Methoden mit generischen Receivern

// Interface definition für non-trivial bound
type I[T any] interface {
	m(T)
}

// TRIVIAL: Generic struct mit trivialem Type Bound (any)
type G[T any] struct {
	_ T
}

// NON-TRIVIAL: Generic struct mit non-trivial Type Bound (Interface constraint)
type G2[T I[T]] struct {
	_ T
}

// Method mit trivialem Type Bound (Receiver ist G[T any])
func (x G[T]) someMethod() {
	// Method implementation
}

func (x G[T]) anotherMethod() {
	// Method implementation
}

// Method mit non-trivial Type Bound (Receiver ist G2[T I[T]])
func (x G2[T]) someMethod() {
	// Method implementation
}

func (x G2[T]) anotherMethod() {
	// Method implementation
}

// Weitere Beispiele:

// TRIVIAL: Empty interface
type SimpleContainer[T interface{}] struct {
	item T
}

func (c SimpleContainer[T]) Get() T {
	return c.item
}

// NON-TRIVIAL: Comparable constraint
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
